# In-Memory Fake Proxmox API (`fakeapi`)

> This document is a design plan — no code is implemented here. It describes a
> `fakeapi` package, an in-memory stand-in for a Proxmox VE cluster that lets
> callers of `go-proxmox-rest` (and this module's own tests) exercise realistic
> multi-node/multi-VM scenarios — attach/detach/resize/delete a disk, list a
> storage's content, watch a node go down — without a live cluster. It plays
> the same role for this module that [`client-go`'s fake
> clientset](https://github.com/kubernetes/client-go/tree/master/kubernetes/fake)
> plays for Kubernetes API consumers.

---

## 1. Motivation

Today this module has two test tiers (§10 of `docs/design.md`):

1. **Unit tests** (`-tags=unit`) — table-driven tests of `internal/params`
   Encode/Decode and per-package request-building logic, with no HTTP
   round-trip.
2. **e2e tests** (`-tags=e2e`, `docs/e2e.md`) — the real thing, opt-in, against
   a live Proxmox cluster reachable via `PVE_E2E_*` env vars.

There's a gap in between: nothing exercises a full multi-call *workflow*
(read config → allocate a volume → attach it → poll the task → verify the
guest's disk list) end-to-end through the resty/HTTP layer, and nothing lets
a **downstream consumer** of this module (an operator, a Terraform provider,
a CLI) unit-test its own orchestration logic against realistic Proxmox
responses without standing up a cluster or hand-rolling `httptest` handlers
per test. `fakeapi` fills that gap: a small, importable, in-memory Proxmox
cluster.

## 2. Goals & Non-Goals

### Goals
- Let a test build a **multi-node cluster** (2-3 nodes is the common case)
  with a handful of pre-seeded VMs and storages, in a few lines.
- Cover the **disk lifecycle** end-to-end through the real client code path:
  allocate/attach, list a storage's content, resize, detach, delete.
- Let a test **fail a node** (mark it offline, make requests to it error) and
  observe how caller code reacts — retries, `IsUnexpected`, HA-style
  failover logic, etc.
- Behave like the real wire protocol closely enough that `go-proxmox-rest`'s
  own decode/error-handling code runs unmodified against it: the `{"data":
  ...}` envelope, Proxmox's error shape, UPID-returning async operations,
  and `internal/params`'s comma-joined-list / stringly-typed-bool quirks.
- Be usable both by this module's own tests (`go test -tags=unit ./...`) and
  by external importers, the same way `client-go/kubernetes/fake` is
  imported by code that never lives in the Kubernetes repo.
- Stay fast and hermetic: in-process, no network beyond `localhost`, safe
  under `t.Parallel()`, no shared global state between `fakeapi.NewCluster`
  calls.

### Non-Goals
- **Not a Proxmox emulator.** It does not run QEMU, does not validate every
  config property-string grammar, and does not reproduce every edge case of
  `PVE::QemuServer`/`PVE::Storage`. It models just enough state — VM config,
  disk inventory, task lifecycle, node membership — to make the client
  library's own request/response contract observable.
- **Not a replacement for e2e.** `docs/e2e.md`'s live-cluster suite remains
  the source of truth for "does this actually work against real Proxmox."
  `fakeapi` can drift from real server behavior; treat a fake-only green
  suite as necessary, not sufficient.
- Full API surface coverage. v1 covers the endpoints named in §7; anything
  else 404s like a real cluster would for an unknown route, so gaps fail
  loudly instead of silently returning zero values.
- Realistic performance/timing simulation (bandwidth limits, IO throttling,
  actual disk allocation).
- **Rate-limiting and load balancing.** The fake does not model Proxmox's
  `429`/`SlowDown` rate-limit response or exercise the root client's
  `retryCondition`/backoff path, and it does not run a per-node listener
  topology to exercise `proxmox.WithRoundRobin`-style connection failover.
  `NewCluster` is a single `httptest.Server`; that's the only entry point
  this fake will ever have. Both are real client behaviors, but they belong
  to a dedicated test harness against a live or simulated multi-listener
  setup, not to this in-memory model.

## 3. Design: why an HTTP-level fake, not an interface mock

`client-go`'s fake clientset works because `kubernetes.Interface` is an
interface — the fake is an alternate implementation swapped in at
construction time, no networking involved. `go-proxmox-rest` has no such
seam at the top: `proxmox.New(...)` returns a concrete `*proxmox.Client`, and
every subpackage (`cluster`, `nodes/qemu`, `nodes/storage`, ...) is wired
directly off it (`docs/design.md` §2, §7) via a package-local `Getter`
interface that only `*proxmox.Client` is ever asked to satisfy. There is
deliberately no root-level interface to swap a fake into — that would mean
maintaining two implementations of `Client.do`/`send`, auth, retries, and the
envelope decoder, and it would let the fake drift from what
`decodeInto`/`internal/params` actually do to a response body.

The seam this module *does* have is the one resty already gives every HTTP
client: the base URL (or transport). `fakeapi` is therefore an
`httptest.Server` wrapping an in-memory cluster model, and a `fakeapi.Cluster`
hands back an ordinary `*proxmox.Client` pointed at it:

```go
cluster := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1", "pve2", "pve3"))
c := cluster.Client(t) // *proxmox.Client, wired to cluster's httptest.Server

// from here on, c is indistinguishable from a client pointed at a real
// cluster — every subpackage's request/response path runs unmodified.
node, err := c.Nodes().Status(ctx, "pve1")
```

This keeps the fake honest: it can only affect what a real Proxmox server
would affect (the bytes on the wire), not reach in and stub out
`decodeInto` or an individual method. Bugs in encode/decode surface against
the fake exactly as they would against a live cluster.

## 4. Package layout

```
fakeapi/
├── fakeapi.go       // Cluster, NewCluster, ClusterOption, Client()/Close()
├── state.go         // in-memory model: node, storage, guest, task records + mutex
├── router.go         // http.Handler wiring: mux, envelope writer, error writer
├── handlers_cluster.go  // /cluster/status, /cluster/resources
├── handlers_nodes.go    // /nodes, /nodes/{node}/status
├── handlers_qemu.go     // /nodes/{node}/qemu/{vmid}/{config,status,resize,unlink,move_disk}
├── handlers_storage.go  // /nodes/{node}/storage[/{storage}/{status,content}]
├── handlers_tasks.go    // /nodes/{node}/tasks/{upid}/status
├── handlers_access.go   // /access/ticket (so password/ticket auth is also testable)
├── tasks.go          // UPID minting + the async task engine (§8)
├── builder.go        // fluent seed-data builders: Node/VM/Storage options
├── errors.go         // Proxmox-shaped error bodies + status codes
└── fakeapi_test.go
```

`fakeapi` imports the root `proxmox` package (to construct `*proxmox.Client`)
and nothing else from this module, so there is no import-cycle risk — it
sits *above* `proxmox` in the dependency graph, symmetrically opposite to how
`docs/design.md` §2 places every resource subpackage *below* it.

## 5. Core types & building a cluster

```go
// NewCluster builds a fake Proxmox cluster and starts its httptest.Server.
// t.Cleanup closes the server automatically.
func NewCluster(t testing.TB, opts ...ClusterOption) *Cluster

type ClusterOption func(*Cluster)

// WithNodes seeds the cluster with the given node names, all online,
// quorate if len(names) > 1. The first name is conventionally the one
// initial requests are described as landing on, but Proxmox has no
// concept of a "primary" node and neither does the fake.
func WithNodes(names ...string) ClusterOption

// WithManualTasks disables the default instant-completion behavior for
// async operations (§8): tasks stay "running" until a test explicitly
// completes them via Cluster.Tasks().Complete/Fail.
func WithManualTasks() ClusterOption

// Cluster is the in-memory Proxmox cluster and its HTTP frontend.
type Cluster struct{ /* ... */ }

// Client returns a *proxmox.Client wired to this cluster over the
// underlying httptest.Server, using a fixed fake API token. Safe to call
// more than once; each call returns an independent *proxmox.Client sharing
// the same backing state (mirroring how multiple real clients can point at
// one cluster).
func (cl *Cluster) Client(t testing.TB) *proxmox.Client

// Node returns a handle for seeding/mutating one node's state. Panics if
// name was not passed to WithNodes (seeding happens at construction; nodes
// are not added dynamically, matching real cluster-join being out of
// scope).
func (cl *Cluster) Node(name string) *Node

// Tasks returns a handle for driving the async task engine under
// WithManualTasks (§8).
func (cl *Cluster) Tasks() *TaskController

// FailNode and RecoverNode simulate a node going down/coming back (§9).
func (cl *Cluster) FailNode(name string, mode FailureMode)
func (cl *Cluster) RecoverNode(name string)
```

`Node` is the seeding/mutation handle for one cluster member:

```go
type Node struct{ /* ... */ }

// AddStorage seeds a storage visible on this node (dir/lvm/zfs/... — Type
// is opaque to the fake beyond being echoed back).
func (n *Node) AddStorage(id, storageType string, opts ...StorageOption) *Storage

// AddVM seeds a QEMU guest on this node with the given config. cfg is a
// *qemu.Config exactly as UpdateConfig/Config already use — no separate
// "fake VM" DTO.
func (n *Node) AddVM(vmid int, cfg *qemu.Config, opts ...VMOption) *VM

type VMOption func(*vmState)

// WithStatus seeds the guest's initial run state (default
// qemu.VMStatusStopped).
func WithStatus(s qemu.VMStatus) VMOption

type StorageOption func(*storageState)

// WithVolume seeds an existing volume in the storage's content list —
// useful for "disk already attached" or "orphaned volume" starting states.
func WithVolume(v storage.Volume) StorageOption

// WithCapacity sets the storage's total/used/avail bytes reported by
// List/Status.
func WithCapacity(total, used, avail int64) StorageOption
```

## 6. In-memory model

`state.go` holds one `sync.Mutex`-guarded struct per cluster:

```go
type clusterState struct {
    mu       sync.Mutex
    nodes    map[string]*nodeState
    tasks    map[string]*taskState // upid -> task
    taskSeq  int64
    quorate  bool
}

type nodeState struct {
    name     string
    online   bool
    failMode FailureMode // "" when healthy
    storages map[string]*storageState
    guests   map[int]*vmState // qemu only for v1 — see §13
}

type storageState struct {
    id, typ  string
    content  []storage.Volume // keyed by VolID
    total, used, avail int64
}

type vmState struct {
    vmid   int
    node   string
    cfg    qemu.Config // scsiN/ideN/... maps ARE the disk inventory
    status qemu.VMStatus
}

type taskState struct {
    upid    string
    node    string
    typ, id string
    running bool
    ok      bool
    errMsg  string
    apply   func() // staged mutation, run on Complete (§8)
}
```

A guest's disks are not a separate model — they live in `vmState.cfg`'s
`SCSI`/`VirtIO`/`IDE`/`SATA`/`Unused` maps, exactly as `qemu.Config` already
represents them (`nodes/qemu/config_types.go`). This is deliberate: it means
"attach a disk" in the fake is the same mutation `UpdateConfig`/
`UpdateConfigAsync` already describe, not a parallel disk-tracking data
structure that could drift from what the client actually sends.

## 7. Endpoint coverage (v1)

| Endpoint | Verb(s) | Backing package | Notes |
|---|---|---|---|
| `/access/ticket` | POST | root `Client.ensureSession` | issues a session the fake also validates on subsequent ticket-auth requests |
| `/cluster/status` | GET | `cluster` | reflects seeded nodes + `FailNode`/quorum (§9) |
| `/cluster/resources` | GET | `cluster` | synthesizes `node`/`storage`/`qemu` entries from state; `type` filter honored |
| `/nodes` | GET | `nodes` | node list with online/offline `status` |
| `/nodes/{node}/status` | GET | `nodes` | 404 semantics for an unknown node name |
| `/nodes/{node}/qemu/{vmid}/config` | GET/PUT/POST | `nodes/qemu` | PUT applies synchronously; POST goes through the task engine (§8) |
| `/nodes/{node}/qemu/{vmid}/status/current` | GET | `nodes/qemu` | |
| `/nodes/{node}/qemu/{vmid}/status/{start,stop,reset,shutdown,reboot,suspend,resume}` | POST | `nodes/qemu` | flips `vmState.status`; always task-backed like real Proxmox |
| `/nodes/{node}/qemu/{vmid}/resize` | PUT | `nodes/qemu` | rewrites the target disk's `size=` property |
| `/nodes/{node}/qemu/{vmid}/unlink` | PUT | `nodes/qemu` | moves the given keys to `unusedN`, mirroring real Proxmox's default unlink behavior |
| `/nodes/{node}/qemu/{vmid}/move_disk` | POST | `nodes/qemu` | reassigns a volume's storage/vmid in state |
| `/nodes/{node}/storage` | GET | `nodes/storage` | |
| `/nodes/{node}/storage/{storage}/status` | GET | `nodes/storage` | |
| `/nodes/{node}/storage/{storage}/content` | GET/POST | `nodes/storage` | POST allocates a synthetic volume, no real bytes written anywhere |
| `/nodes/{node}/storage/{storage}/content/{volume}` | GET/PUT/DELETE | `nodes/storage` | DELETE is unconditional — see §13 |
| `/nodes/{node}/tasks/{upid}/status` | GET | `nodes/tasks` | |
| `/nodes/{node}/tasks/{upid}` | DELETE | `nodes/tasks` | `Stop` — marks a manual-mode task failed with `"task stopped"` |

Everything else 404s. This is intentional: silently succeeding with zero
values for an unimplemented endpoint would make a test pass for the wrong
reason. Extending coverage means adding a handler file plus a state slice,
following the existing files as a template — the same "extend without
modifying" seam `docs/design.md` §7 describes for the real client.

## 8. Async tasks / UPIDs

Every write that's async in real Proxmox (`status/*`, `resize`,
`move_disk`, `UpdateConfigAsync`, `storage content` delete with no `delay`)
returns a UPID in the fake too, in the same
`UPID:{node}:{pid}:{pstart}:{starttime}:{type}:{id}:{user}:` shape real
Proxmox uses, so code that parses or logs a UPID sees realistic input.

Two modes, selected once at `NewCluster` time:

- **Instant (default).** The state mutation and the task's completion both
  happen synchronously inside the handler, before the HTTP response is
  written. `Tasks().Status(ctx, node, upid)` immediately reports
  `StateStopped` / `ExitStatus: "OK"`. This is the right default for tests
  whose subject is "did the right mutation happen," not "does my polling
  loop work" — which is most tests exercising the disk lifecycle in §1.
- **Manual (`WithManualTasks`).** The handler stages the mutation (as
  `taskState.apply`) and returns a UPID for a task left `running`. The test
  drives it explicitly:

  ```go
  upid, err := c.Nodes().Qemu().Resize(ctx, "pve1", 100, &qemu.ResizeOptions{
      Disk: "scsi1", Size: "+10G",
  })
  // upid's task is "running"; the resize has NOT been applied to state yet.

  cluster.Tasks().Complete(upid) // applies the staged mutation, flips to StateStopped/"OK"
  // or: cluster.Tasks().Fail(upid, "some error") // discards the mutation, ExitStatus = the message
  ```

  This is what makes retry logic, "poll until done" helpers, and
  `Tasks().Stop` genuinely testable — the state transition is under the
  test's control instead of always winning the race instantly.

## 9. Simulating node failure

`Cluster.FailNode(name string, mode FailureMode)` and `RecoverNode(name
string)` are the two calls a test needs. `FailureMode` covers the two
shapes a node outage actually takes against this client:

```go
type FailureMode int

const (
    // FailureUnreachable: every request scoped to this node (any
    // /nodes/{name}/... path) returns HTTP 596 with a Proxmox-shaped
    // error body — the status code real Proxmox's own inter-node API
    // proxy returns when it cannot reach a cluster member. IsUnexpected
    // reports true for it; IsNotFound does not.
    FailureUnreachable FailureMode = iota

    // FailureConnRefused: the fake closes the TCP connection outright
    // instead of writing an HTTP response, so callers see a transport
    // error (no *proxmox.APIError to unwrap) — for exercising code paths
    // that specifically handle network errors distinctly from API
    // errors.
    FailureConnRefused
)
```

`FailNode` also updates the node's entry in `/cluster/status` and
`/cluster/resources` (`Online: 0`, `Status: "offline"`) so code that watches
cluster-wide state (rather than polling the node directly) observes the
outage too, and recomputes `Status.Quorate` from the surviving online count
— useful for exercising HA-adjacent logic that gates on quorum before acting.
Guests already running on a failed node keep their last-known `vmState`
(mirroring how a real node's `pvestatd` simply stops reporting rather than
guests visibly changing state) — any request explicitly targeting that node
still fails per `mode`, but the guest's entry in `/cluster/resources` is
left at its last-seen values, not synthetically zeroed.

```go
cluster := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1", "pve2", "pve3"))
c := cluster.Client(t)

cluster.FailNode("pve3", fakeapi.FailureUnreachable)

_, err := c.Nodes().Status(ctx, "pve3")
// err is a *proxmox.APIError with StatusCode 596; proxmox.IsUnexpected(err) == true

status, _ := c.Cluster().Status(ctx)
// status.Quorate reflects 2/3 nodes online; the "pve3" entry in status.Nodes
// has Online == 0

cluster.RecoverNode("pve3")
_, err = c.Nodes().Status(ctx, "pve3") // succeeds again
```

## 10. Error responses & the `Is*` predicates

`errors.go` writes bodies in the same shape `errors.go` (root package)
expects to parse — `{"data": null, "errors": {...}, "message": "..."}` —
with the status code chosen to exercise the matching predicate:

| Scenario | Status | `proxmox.Is*` |
|---|---|---|
| Unknown node/vmid/storage/volume | 404 | `IsNotFound` |
| `FailureUnreachable` | 596 | `IsUnexpected` |
| Malformed/missing required param (e.g. `Resize` with no `disk`) | 400 | neither — same as today, caught client-side before the request is even sent |

Rate-limiting (`429`/`SlowDown`, `IsRateLimited`) is explicitly out of scope
— see the Non-Goals in §2.

## 11. Worked example

```go
func TestDiskLifecycle(t *testing.T) {
    cluster := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1", "pve2", "pve3"))

    pve1 := cluster.Node("pve1")
    pve1.AddStorage("local-lvm", "lvm", fakeapi.WithCapacity(500<<30, 100<<30, 400<<30))
    pve1.AddVM(100, &qemu.Config{
        Name:  "web-01",
        Cores: 2, Memory: "2048",
        SCSI: map[int]string{0: "local-lvm:vm-100-disk-0,size=20G"},
    }, fakeapi.WithStatus(qemu.VMStatusRunning))

    c := cluster.Client(t)
    ctx := context.Background()

    // attach: allocate a volume, then wire it into the guest config
    volid, err := c.Nodes().Storage().Content().Create(ctx, "pve1", "local-lvm", &storage.CreateVolumeOptions{
        Filename: "vm-100-disk-1", VMID: 100, Size: "10G", Format: "raw",
    })
    require.NoError(t, err)

    cfg, err := c.Nodes().Qemu().Config(ctx, "pve1", 100, nil)
    require.NoError(t, err)
    cfg.SCSI[1] = volid + ",size=10G"
    _, err = c.Nodes().Qemu().UpdateConfigAsync(ctx, "pve1", 100, cfg)
    require.NoError(t, err)

    // list disks in node storage
    vols, err := c.Nodes().Storage().Content().List(ctx, "pve1", "local-lvm", &storage.ContentListOptions{VMID: 100})
    require.NoError(t, err)
    require.Len(t, vols, 2) // disk-0 (seeded) + disk-1 (just attached)

    // resize
    _, err = c.Nodes().Qemu().Resize(ctx, "pve1", 100, &qemu.ResizeOptions{Disk: "scsi1", Size: "+5G"})
    require.NoError(t, err)

    // detach
    err = c.Nodes().Qemu().Unlink(ctx, "pve1", 100, &qemu.UnlinkOptions{IDList: []string{"scsi1"}})
    require.NoError(t, err)

    // delete the now-unused volume
    _, err = c.Nodes().Storage().Content().Delete(ctx, "pve1", "local-lvm", volid, 0)
    require.NoError(t, err)

    // node failure: pve1 goes away mid-workflow
    cluster.FailNode("pve1", fakeapi.FailureUnreachable)
    _, err = c.Nodes().Qemu().Status(ctx, "pve1", 100)
    require.True(t, proxmox.IsUnexpected(err))
}
```

## 12. Usage in unit tests vs e2e

`fakeapi`-based tests are **unit tests** (`-tags=unit`, `make unit`) — they
have no external dependency and run in every `go test ./...` invocation. A
natural home is `tests/fake/` (mirroring `tests/e2e/`'s per-module layout)
for larger workflow tests, alongside `fakeapi_test.go` in the package itself
for testing the fake's own handlers. Nothing about `fakeapi` requires the
`unit` build tag itself — it's ordinary Go code any package can import — but
tests that *use* it to validate this module's own client code should
live under the `unit` tag for consistency with `docs/design.md` §10 and the
existing `make unit`/`make test` targets.

`docs/e2e.md`'s suite is unaffected and remains required for release
confidence — see the Non-Goals in §2.

## 13. Open questions / future milestones

- **LXC and cluster/HA state.** v1 seeds only QEMU guests
  (`nodeState.guests`) and the plain node/quorum model. LXC containers,
  `cluster/ha` groups/rules, and `cluster/jobs` are natural v2 additions
  once QEMU coverage proves the pattern out — same shape, new state slices
  and handler files per §4/§7.
- **Volume-in-use delete: resolved as unconditional, no guard.** `Content().Delete`
  in real Proxmox (`PVE::Storage::vdisk_free`) is guest-agnostic — storage
  plugins have no visibility into `qemu`/`lxc` configs, so deleting a volume
  that's still wired into a guest's `scsiN=`/`ideN=`/... config succeeds and
  leaves that config with a dangling reference (the well-known "deleted the
  disk from Datacenter > Storage but the VM won't start" footgun). This is
  confirmed **not** gated by `Protected` — `tests/e2e/nodes/storage` shows
  DirPlugin rejects `Protected` on anything but a backup-type volume, so it
  can't be the in-use mechanism for a disk image. `fakeapi`'s `Content.Delete`
  therefore does not consult `vmState.cfg` at all: it removes the volume from
  `storageState.content` unconditionally, even if some guest's config still
  names it, matching the real footgun rather than adding a safety net
  Proxmox itself doesn't have. Not yet e2e-verified end-to-end (attach →
  delete-out-from-under-the-guest → observe the dangling reference); worth a
  follow-up e2e case before this is fully load-bearing.

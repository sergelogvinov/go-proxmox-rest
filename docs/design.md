# go-proxmox-rest — Design Document

> A Go client library for the [Proxmox VE REST API](https://pve.proxmox.com/pve-docs/api-viewer/).

This document describes the architecture and conventions of the module. The goal is
to provide a **type-safe, idiomatic Go** wrapper around the Proxmox VE API, built on
top of [`go-resty/resty`](https://github.com/go-resty/resty), with a **fluent
resource-chaining API** in the style of the Kubernetes client-go pattern:

```go
client.Cluster().Options().Get(ctx)
client.Nodes().Get(ctx, "pve")
```

---

## 1. Goals & Non-Goals

### Goals
- Full, typed coverage of the Proxmox VE API surface (`access`, `cluster`, `nodes`,
  `pools`, `storage`, `version`).
- Thin, predictable mapping between Go structs and the Proxmox wire format.
- Clean separation of concerns:
  - **HTTP transport** (resty) — low level, reusable.
  - **Resource builders** — fluent chaining (`Client.Cluster()....`).
  - **Data layer** — DTO/response structs and encode/decode helpers.
- Graceful handling of Proxmox's unusual response envelope (see §4).
- Backward compatibility with the wire format so existing Proxmox scripts remain sane.

### Non-Goals (v1)
- Implement every single endpoint in the first release — the design must make it
  trivial to add more.
- A full code generation toolchain from the OpenAPI spec (optional later milestone).
- Async/streaming execution of cluster tasks (may be added on top via the `UPID`/task API).

---

## 2. Module Layout

The package mirrors the Proxmox API resource tree. Each top-level API section is its
own Go package. The `proxmox` root package owns the `Client` and shared types.

```
go-proxmox-rest
├── go.mod                     // module github.com/sergelogvinov/go-proxmox-rest
├── client.go                  // Client, api-client (HTTP), TLS/auth, fluent entry points
├── resource.go                // shared resource primitives (resource.layer)
├── types.go                   // shared types (UPID, Task, errors, enums)
├── options.go                 // request/query options builders
├── access/
│   ├── client.go              // Access().User(), Access().Group(), Access().Role()...
│   ├── user.go                // user DTOs + encode/decode
│   └── ...
├── cluster/
│   ├── resources.go           // Cluster().Resources()
│   ├── options.go
│   ├── ha/                    // cluster/ha/ — /cluster/ha/* (groups, rules)
│   │   ├── groups.go          // Cluster().HA().Groups()
│   │   ├── rules.go           // Cluster().HA().Rules()
│   │   └── types.go
│   └── ...
├── nodes/
│   ├── client.go              // Nodes().Get(ctx, "node")
│   ├── qemu.go                // Nodes().<node> .Qemu().Vm(id)... (future)
│   ├── ...
├── pools/
├── storage/
└── version/
```

### Dependency direction
```
proxmox (root)   ->  resty
      ^
      |
access/cluster/nodes/pools/storage/version  ->  proxmox (root)
      ^
      |
cluster/ha  ->  proxmox (root)   // not -> cluster
```

Child packages depend **only** on the root package — never on each other — so the
dependency graph stays acyclic and each section can be developed independently.

### Two levels of nesting

Proxmox nests some API sections one level deeper than the top-level resource tree
(e.g. `/cluster/ha/groups`, `/cluster/ha/rules`). Where a section's endpoints live
under a path with their own sub-resources and enough surface area to earn their own
DTOs/encode/decode files, the module layout mirrors that with a second directory
level (`cluster/ha/`) rather than piling more types into the parent package
(`cluster/`). This is a size/surface judgment call, not a rule applied to every
nested path — a handful of fields folded into the parent's `types.go` (as
`cluster.Options` is for `/cluster/options`) stays in the parent package.

The dependency rule from §2 generalizes unchanged: `cluster/ha` depends only on the
root package (it defines its own `Getter` interface, satisfied structurally by
`*proxmox.Client`, exactly like `cluster`, `pools`, etc.) — **not** on `cluster`,
even though `cluster/ha` lives inside the `cluster/` directory. The only edge
connecting the two is the parent wiring the child in, the same lazy-construction
seam root uses for its direct children:

```go
// cluster/status.go
func (c *Client) HA() *ha.Client { return ha.New(c.client) }
```

Naming inside a nested package drops the parent's name as a prefix — `ha.Group`,
not `ha.HAGroup` — the same stutter-avoidance already used for `cluster.Options`,
`cluster.Status`, and `cluster.Resource` rather than `cluster.ClusterOptions` etc.

---

## 3. Resty Client Setup

`proxmox.New(...)` builds the underlying resty `*resty.Client` and wires
authentication, retries, timeouts, and logging.

```go
// All fields are private (lowercase) — they can only be set through the
// functional options (see below), never via a struct literal.
type ClientConfig struct {
    baseURL          string        // full API endpoint, e.g. https://10.0.0.1:8006/api2/json
    token            string        // root@pam!token / user@realm!tokenid (PVE API token)
    secret           string        // API token secret
    insecure         bool          // allow self-signed certs
    timeout          time.Duration // request timeout (default 5m)
    retryCount       int           // number of auto-retries (default 3)
    retryWaitTime    time.Duration // initial backoff wait per retry (default 500ms)
    retryMaxWaitTime time.Duration // cap on backoff wait (default 30s)
    userAgent        string        // custom User-Agent, e.g. "go-proxmox-rest/v0.1.0"
    logger           resty.Logger  // resty logger (default none/severity based)
    proxy            string        // optional proxy URL
    caCert           string        // path to a PEM CA bundle to trust
    lb               resty.LoadBalancer // optional; when set, replaces baseURL
}

// New builds a Client from a zero-value ClientConfig and a set of options.
func New(cfg ClientConfig, opts ...Option) (*Client, error)

// ToRESTConfig returns a deep copy of the config, safe to mutate without
// affecting the original. Useful to derive per-request or per-cluster configs.
// Note: private fields only; copies are made by value, so all fields are
// independent.
func (c ClientConfig) ToRESTConfig() ClientConfig
```

All `ClientConfig` fields are **private** and only set through options. The struct
is never constructed directly with a literal — the zero value is passed to `New`
and the functional options populate it:

```go
// options.go
type Option func(*ClientConfig)

func WithURL(u string) Option // path component (e.g. /api2/json) is saved as basePath
func WithBasePath(p string) Option // explicit API path prefix, overrides the URL path
func WithTokenAuth(tokenID, secret string) Option
func WithPasswordAuth(username, password, realm string) Option
func WithTimeout(d time.Duration) Option
func WithRetryCount(n int) Option
func WithRetryWaitTime(d time.Duration) Option
func WithRetryMaxWaitTime(d time.Duration) Option
func WithUserAgent(ua string) Option
func WithInsecure(skip bool) Option
func WithCACert(path string) Option // path to a PEM CA bundle to trust
func WithProxy(p string) Option
func WithLogger(l resty.Logger) Option

// Load balancer (resty v3): overrides the single BaseURL. Use one of these or
// `WithLoadBalancer` to plug a custom resty.LoadBalancer implementation.
func WithRoundRobin(urls ...string) Option          // NewRoundRobin
func WithWeightedRoundRobin(hosts []resty.Host) Option // NewWeightedRoundRobin
func WithSRVWeightedRoundRobin(service, proto, domain, scheme string) Option // NewSRVWeightedRoundRobin
func WithLoadBalancer(lb resty.LoadBalancer) Option // any custom implementation
```

Precedence is `New(cfg, opts...)` → each `Option` mutates a copy of `cfg` (last one
wins). Since fibelds are private, options are the **only** way to populate config.
Deriving a mutable snapshot for reuse is done via `cfg.ToRESTConfig()`:

```go
// usage
c, err := proxmox.New(
    proxmox.ClientConfig{}, // zero value; everything set via options
    proxmox.WithURL("https://pve:8006/api2/json"),
    proxmox.WithTokenAuth(tokenID, secret),
    proxmox.WithCACert("/etc/ssl/pve-ca.pem"),
    proxmox.WithTimeout(60*time.Second),
    proxmox.WithUserAgent("proxmox-ops/v1.2.0"),
)

// derive a mutable copy for a per-cluster tweak
cfg := proxmox.ClientConfig{} // read from elsewhere (e.g. credential store)
snapshot := cfg.ToRESTConfig()
// ...snapshot is privately held; options still required to mutate it
```

The `apiClient` wraps resty and exposes **internal** methods used by all resource
packages:

```go
type apiClient struct{ rc *resty.Client }

// Do executes a typed request and decodes the payload into out.
func (a *apiClient) do(ctx context.Context, method, path string, opts ...ReqOption) error
```

Key resty wiring (resty v3 API):

- **Base URL** — `SetBaseURL(cfg.BaseURL)`; if omitted, derived by normalizing a host
  into `https://<host>:8006/api2/json`.
- **Load balancer** — when a `With*LoadBalancer`/`WithLoadBalancer` option is set,
  `SetLoadBalancer(cfg.lb)` overrides the single `BaseURL`. Built-in options map
  directly to resty's helpers: `WithRoundRobin` → `resty.NewRoundRobin`,
  `WithWeightedRoundRobin` → `resty.NewWeightedRoundRobin`,
  `WithSRVWeightedRoundRobin` → `resty.NewSRVWeightedRoundRobin`, and
  `WithLoadBalancer` accepts any custom `resty.LoadBalancer`.
- **Auth**: resty `SetAuthToken()` with `PVEAPIToken=<token>=<secret>`; optionally
  `--with-header` alternative for ticket auth on legacy setups.
- **Retry**: `SetRetryCount(cfg.RetryCount)` + `SetRetryWaitTime(cfg.RetryWaitTime)` +
  `SetRetryMaxWaitTime(cfg.RetryMaxWaitTime)` with a predicate that only retries on
  the explicit Proxmox `SlowDown` (task queue) response, plus 5xx and connectivity.
- **Logging**: `SetLogger(cfg.Logger)`; sensitive headers (token/secret) masked.
- **Timeout**: `SetTimeout(cfg.Timeout)` from config.
- **TLS**: `SetTLSClientConfig(...)` built from `cfg` — if `WithCACert` was set, the
  PEM bundle is appended via `SetRootCertificate`/`tls.Config.RootCAs`; otherwise
  `InsecureSkipVerify: cfg.Insecure`.
- **User-Agent**: `SetHeader("User-Agent", cfg.UserAgent)`.

> Authentication approach is captured in its own ADR (see §9) — both ticket-based
> login (for node endpoints) and API-token auth flow must be supported.

---

## 4. Proxmox Response Envelope

Proxmox wraps **almost every** response in an envelope. The design encodes this as a
single generic unwrap rather than per-endpoint structs.

```jsonc
// success
{ "data": ... }

// error
{
  "data": null,
  "errors":  { "field": "error message", ... },
  "message": "parameter verification failed"
}
```

`data` is commonly `null`, a scalar, an object, a list, or even a **string with
embedded line breaks** (e.g. `vzdump` logs, certificate bundles). To keep the DTOs
clean, the root package provides decoding helpers:

```go
// parseResponse quickly unwraps the { "data": ... } envelope.
func decodeInto[T any](b []byte) (T, error)
```

- `data: null` → zero value.
- `data: string` that is actually JSON → auto-unmarshalled.
- Unknown fields are **ignored** by default, so the API can grow without breaking us.

The wire `data` field is **snake_case**; Go fields are `camelCase` with struct tags
(see §6).

---

## 5. The Fluent Resource Chain

This is the core developer-facing API. It is inspired by
[client-go](https://github.com/kubernetes/client-go): start from the `Client`, chain
down the resource tree, then invoke an **HTTP verb method** that takes
`context.Context` and returns a typed value.

```go
// root entry points
func (c *Client) Cluster() *cluster.Client
func (c *Client) Nodes()   *nodes.Client
func (c *Client) Access()  *access.Client
func (c *Client) Pools()   *pools.Client
func (c *Client) Storage() *storage.Client
func (c *Client) Version() *version.Client
```

### The `resource` primitive

Every terminal resource is backed by a small internal generic `resource` type that
carries the `*Client` and the **dynamic portion** of the path, plus optional filters.

```go
package proxmox

type resource struct {
    client *Client
    parts  []string // dynamic path segments
    name   string   // last segment (node id, vm id, storage id, ...)
}
```

This lets a builder hold state like "I am node `pve`" without re-parsing the path.

### HTTP verb methods

All verb methods accept `ctx context.Context` **first**:

| Method | Meaning                                   | Example                                   |
|--------|-------------------------------------------|-------------------------------------------|
| `Get`  | `GET <path>` (also returns created/derived data) | `c.Nodes().Get(ctx, "pve")`        |
| `Create` | `POST <path>` (with body)              | `c.Pools().Create(ctx, pool)`       |
| `Update` | `PUT <path>` (with body)              | `c.Cluster().Options().Update(ctx, opts)` |
| `Delete` | `DELETE <path>`                        | `c.Nodes().Delete(ctx, "pve")`      |
| `Patch` | `PATCH <path>` (partial)              | `c.Cluster().Options().Patch(ctx, ...)` |

Signature pattern:

```go
// options.go
func (o *optionsResource) Get(ctx context.Context) (*cluster.Options, error)
func (o *optionsResource) Update(ctx context.Context, opts *cluster.Options) (*cluster.Options, error)
```

### Working example

```go
// skip load balancing — single cluster. See below for a multi-PVE-host setup.
c, err := proxmox.New(
    proxmox.ClientConfig{},
    proxmox.WithTokenAuth(tokenID, secret),
    proxmox.WithCACert("/etc/ssl/pve-ca.pem"), // or proxmox.WithInsecure(true)
    proxmox.WithTimeout(60*time.Second),
    proxmox.WithRetryCount(3),
    proxmox.WithUserAgent("proxmox-ops/v1.2.0"),
)
if err != nil { log.Fatal(err) }

ctx := context.Background()

// read a node
node, err := c.Nodes().Get(ctx, "pve")
if err != nil { log.Fatal(err) }

// list cluster resources (opt-in type filter passed to Get; "" is unfiltered)
res, err := c.Cluster().Resources().Get(ctx, cluster.ResourceTypeVM)
```

With a **load balancer**, point the same `Client` at several Proxmox nodes and let
resty pick the target per-request (round-robin or weighted). The resource chain is
unchanged:

```go
// round-robin across a cluster of PVE nodes
c, err := proxmox.New(
    proxmox.ClientConfig{},
    proxmox.WithTokenAuth(tokenID, secret),
    proxmox.WithRoundRobin(
        "https://pve1:8006/api2/json",
        "https://pve2:8006/api2/json",
        "https://pve3:8006/api2/json",
    ),
)
if err != nil { log.Fatal(err) }

// or weighted, when nodes differ in capacity
c, err = proxmox.New(
    proxmox.ClientConfig{},
    proxmox.WithTokenAuth(tokenID, secret),
    proxmox.WithWeightedRoundRobin([]resty.Host{
        {BaseURL: "https://pve1:8006/api2/json", Weight: 50},
        {BaseURL: "https://pve2:8006/api2/json", Weight: 30},
        {BaseURL: "https://pve3:8006/api2/json", Weight: 20},
    }),
)

// resource semantics are identical whichever target strategy is chosen
node, err = c.Nodes().Get(ctx, "pve")
if err != nil { log.Fatal(err) }
```

Chaining is **only** structural — semantics live in the final verb call, keeping the
builder methods side-effect free and easy to reason about.

---

## 6. Struct ↔ Wire Mapping (DTO Layer)

Proxmox uses **snake_case** JSON. Go uses **camelCase** fields. Every DTO carries
explicit struct tags, mirroring the API viewer's field names.

```go
// cluster/options.go
type Options struct {
    Keyboard string `json:"keyboard,omitempty"`
    Language string `json:"language,omitempty"`
    ...
}
```

### Encode helpers
Proxmox frequently expects flat, form-encoded **query/body parameters** (e.g.
`POST /nodes/{node}/qemu` takes a flat param list) rather than a nested object. The
root package provides generic encoder/validator helpers reused across DTOs:

```go
// Use struct tags to map Go fields to API param names.
type PostTag struct {
    Tag            string `url:"tag"`
    IsTemplate     bool   `url:"istemplate"`
    Force          *bool  `url:"force,omitempty"`
    Password       string `url:"password,omitempty"`
    Revert         *bool  `url:"revert,omitempty"`
}

func EncodeParams(t any) (url.Values, error) // tag-driven, skip zero/nil, handle nested
```

Rules:
- `omitempty`-style behavior via **pointer + `nil`** so a value of `false`/`0` is still
  serialized when meaningful.
- Scalars remain scalars; slices serialize as comma-joined strings or repeated keys
  depending on the endpoint.
- Endpoint-specific quirks go in a per-package `encode.go`.

### Read (`X`) vs. write (`XOptions`) structs: when to merge

Every resource has a read shape (returned by GET) and a write shape (accepted by
POST/PUT). Before adding a new resource, check whether these two shapes should be one
Go type or two — don't default to splitting them without checking, and don't merge
them just because the field lists look similar at a glance.

**Default: two separate types, `X` (read) and `XOptions` (write).** This is the shape
for every multi-item, key-addressed resource (`List`/`Get`/`Create`/`Update`/`Delete`)
— e.g. `pools.Pool`/`pools.Options`, `ha.Group`/`ha.GroupOptions`,
`ha.Rule`/`ha.RuleOptions`, `firewall.Alias`/`firewall.AliasOptions`,
`firewall.Group`/`firewall.GroupOptions`, `firewall.IPSet`/`firewall.IPSetOptions`,
`firewall.Rule`/`firewall.RuleOptions`. Keep them separate whenever any of these hold:
- Update needs to distinguish "leave unchanged" from "explicitly clear to zero/empty"
  — that requires pointer fields (`*string`, `*bool`, ...) on the write side, which
  would force needless nil-checks on the read side for no benefit.
- Create and Update don't require exactly the same fields (e.g. `RuleOptions.Type`/
  `Action` are required on both, but most other fields are pointer-optional).
- Either side has fields the other has no use for: write-only knobs (`Rename`,
  `Delete []string`), or read-only/server-computed fields (`Digest` as returned by
  GET, `Pos`, `IPVersion`).

**Exception: merge into one type for a cluster-wide singleton** — a resource with
exactly one instance and no Create/Delete, only GET/PUT (currently `cluster.Options`
and `firewall.Options`, i.e. the `.../options` config endpoints). Use one struct with
dual `json:"...,omitempty"` + `url:"...,omitempty"` tags per field, non-pointer fields
relying on the params encoder's "skip the zero value" rule, and a `Delete []string`
field for explicit resets. There's no Create/Update field-set asymmetry to reconcile
(PUT is the only write verb) and no per-item pointer "clear" semantics needed beyond
what `Delete` already covers, so a second type would be a field-for-field duplicate.

**The check, concretely:** list the read fields and the write fields side by side. Only
merge when they're identical *and* the resource is a GET/PUT singleton. A near match on
a key-addressed resource is not enough — e.g. `firewall.Alias` (`Name`, `CIDR`,
`Comment`, `IPVersion`, `Digest`) vs. `firewall.AliasOptions` (`Name`, `CIDR`,
`Comment *string`, `Rename *string`, `Digest`) look close, but `IPVersion` is
read-only, `Rename` is write-only, and `Comment` needs pointer semantics on write —
merging would leak write-only/read-only fields across both directions and lose the
unchanged-vs-cleared distinction on `Comment`. As of this writing, the merge applies to
exactly one subtree (the `.../options` singletons); every other resource keeps the
two-type shape.

**`readonly`/`writeonly` tag modifiers (a narrower tool, not a blanket merge-enabler).**
`internal/params` tags support `url:"name,readonly"` and `url:"name,writeonly"` (see
`internal/params/tag.go`'s `parseTag`): a `readonly` field is populated by `Decode` but
never sent by `Encode`, no matter its value; a `writeonly` field is the mirror image.
This solves *one* of the three reasons listed above for keeping types separate — fields
that only make sense on one side (a write-only `Rename`, a read-only `Digest`/`Pos`/
`IPVersion`) can now coexist in a single struct without a read call accidentally
populating a write-only field, or a write call built by copying a fetched struct
accidentally re-sending a read-only one. It does **not** solve the other two: it can't
give a plain field pointer "clear vs. unchanged" semantics, and it can't make Create and
Update require different fields. Concretely, this is why `firewall.Alias` still can't
merge with `AliasOptions` even with these modifiers available: `IPVersion`/`Rename`
could be tagged `readonly`/`writeonly` easily enough, but `Comment` would still need to
be a pointer to support explicit-clear on Update (Proxmox's alias `PUT` has no `delete`
list to fall back on), which pushes nil-checks onto every read-path caller — the actual
blocker was never the field-name overlap. Reach for these modifiers only when a
resource's read/write shapes are identical *except* for a handful of one-sided fields,
none of which need pointer clear-semantics.

### Decode helpers
```go
// Generic decode: unwrap envelope, then map snake_case JSON → Go struct.
// Because some "data" payloads are newline-delimited text (vzdump logs),
// a raw decoder is provided per resource when necessary.
```

---

## 7. Client seams: extend without modifying

Every resource package exposes a tiny internal interface that the root `Client` uses,
so new sections can be added without touching existing code:

```go
package nodes
type Client struct { *proxmox.Client }
func New(c *proxmox.Client) *Client { return &Client{c} }

func (c *Client) Get(ctx context.Context, node string) (*Node, error) { ... }
```

The root `Client` lazily constructs them:

```go
func (c *Client) Nodes() *nodes.Client { return nodes.New(c) }
```

This keeps the root package small and lets each API section grow independently.

---

## 8. Error Handling

Proxmox errors are returned as **typed** Go errors on every verb method:

```go
type APIError struct {
    StatusCode int
    Message    string
    Errors     map[string]string // per-field validation errors
}

func (e *APIError) Error() string { ... }
```

- resty transport errors (network, TLS, timeout) are wrapped with `fmt.Errorf`/
  `errors.Join` so callers can inspect both the HTTP and Unwrap interfaces.
- `context.Canceled` / `context.DeadlineExceeded` propagate unmodified.

Helper predicates for callers:
```go
func IsNotFound(err error) bool
func IsRateLimited(err error) bool   // maps to Proxmox "SlowDown"
func IsUnexpected(err error) bool
```

---

## 9. Authentication (ADR)

Two mechanisms must be supported:

1. **API Token** — the recommended, long-lived approach used by CI/automation.
   `Authorization: PVEAPIToken=<tokenid>=<secret>`.
2. **Ticket/Password** — session cookie auth used by web UI and some node endpoints.
   `POST /access/ticket` returns the `PVEAuthCookie` we attach to subsequent requests.

Wire both behind a single resty `PreRequestHook` (or `SetAuthScheme`/`SetAuthToken`)
chosen at `New(...)` time. Tokens are **never logged or serialized** into structs.

---

## 10. Testing & Mocking

- **VCR-style mocking**: record real Proxmox responses into fixtures
  (`testdata/`) and replay them; avoids needing a live cluster.
- resty's `SetTransport` round-tripper makes this trivial — inject a `httptest` server
  or a recording transport in tests.
- Table-driven tests for Encode/Decode helpers and error mapping.

---

## 11. Future Milestones

1. **v0.1** — core `Client`, resty wiring, auth, envelope unwrap, error types, and
   `cluster` + `nodes` read paths.
2. **v0.2** — full `access`, `storage`, `pools`, `version`; write paths
   (`Create/Update/Delete`).
3. **v0.3** — QEMU/LXC VM lifecycle under `nodes` (start/stop/status/snapshots),
   task/UPID tracking for async operations.
4. **Later** — optional codegen from a vendored OpenAPI spec.

---

## Appendix: Reference — Top-level resource tree

| Segment    | Description                                          |
|------------|------------------------------------------------------|
| `access`   | Authentication, users, groups, roles, tokens, ACLs   |
| `cluster`  | Cluster-wide config, resources, HA, backup, logs     |
| `nodes`    | Per-node status, config, QEMU/LXC, storage, network  |
| `pools`    | Resource pools                                       |
| `storage`  | Storage config & content                             |
| `version`  | Server version info                                  |
# End-to-End (E2E) Integration Test Plan

> This document describes how we will test the `go-proxmox-rest` client
> against a **real Proxmox VE cluster**. It is a plan only — no code is
> implemented here. The goal is to exercise the public API surface
> (`list → get → create → get → update → get → delete → list`) against live
> endpoints and verify the client behaves correctly.

---

## 1. Goals & Non-Goals

### Goals
- Verify the client works against a **real** Proxmox VE cluster (not a mock).
- Exercise the full CRUD lifecycle for each module: `list`, `get`, `create`,
  `get` (verify create), `update`, `get` (verify update), `delete`, `list`
  (verify delete).
- Validate authentication paths: **API token** and **password/ticket**.
- Validate error handling: `IsNotFound`, `IsRateLimited`, `IsUnexpected`.
- Validate the response envelope unwrapping (`{ "data": ... }`) and typed
  struct decoding for each module.
- Provide a repeatable, documented way to run the suite against any cluster.

### Non-Goals
- Testing every single endpoint/parameter combination.
- Performance/load testing.
- Testing against a mock or `httptest` server (that is covered by unit tests).
- Destructive testing of pre-existing cluster resources — the suite creates
  and cleans up its **own** uniquely-named resources only.

---

## 2. Test Environment & Prerequisites

The suite runs against a live cluster. It must be **opt-in** so it never runs
during normal `go test ./...` (unit) runs.

### 2.1 Cluster Requirements
- A reachable Proxmox VE node (e.g. `https://pve.example.com:8006`).
- A user with sufficient privileges to create/update/delete the resources the
  suite touches (e.g. `root@pam` or a dedicated `e2e@pve` user with
  `Administrator` on `/`).
- For storage tests: a node with a writable filesystem path available for a
  temporary `dir`-type storage.

### 2.2 Credentials
Two auth modes must be supported and selectable via env vars:

| Env var | Purpose |
|---------|---------|
| `PVE_E2E_URL` | Base URL, e.g. `https://pve:8006/api2/json` |
| `PVE_E2E_TOKEN_ID` | API token ID, e.g. `root@pam!e2e` |
| `PVE_E2E_TOKEN_SECRET` | API token secret |
| `PVE_E2E_USERNAME` | Username for password/ticket auth, e.g. `root@pam` |
| `PVE_E2E_PASSWORD` | Password for password/ticket auth |
| `PVE_E2E_INSECURE` | `1` to skip TLS verification (self-signed certs) |
| `PVE_E2E_CA_CERT` | Path to a CA bundle to trust |
| `PVE_E2E_NODE` | Node name to scope node-dependent tests (e.g. `pve`) |
| `PVE_E2E_PREFIX` | Prefix for created resource names (default `e2e-`) |
| `PVE_E2E_PARALLEL` | `true` to run module subfolders in parallel (default `false`) |
| `PVE_E2E_CLEANUP_ON_FAILURE` | `false` to keep resources on test failure (default `true`) |

### 2.3 Gating
- The suite is gated behind a build tag, e.g. `//go:build e2e`, so it is
  excluded from normal unit-test runs.
- A helper `TestMain` (or a `setup` helper) reads the env vars, builds the
  client, and **skips** the whole suite (`t.Skip`) when `PVE_E2E_URL` is
  unset.
- Run command (documented in `Makefile`):
  ```
  go test -tags=e2e ./tests/e2e/... -v
  ```

---

## 3. Connection Client Design

A single shared connection helper lives at the top of the e2e suite and is
reused by every module subfolder. It is responsible for:

1. **Reading config** from the env vars in §2.2.
2. **Building the client** via `proxmox.New(cfg, opts...)`.
3. **Choosing auth mode**:
   - If `PVE_E2E_TOKEN_ID` is set → `proxmox.WithTokenAuth(id, secret)`.
   - Else if `PVE_E2E_USERNAME`/`PVE_E2E_PASSWORD` are set →
     `proxmox.WithPasswordAuth(user, pass)`.
   - Else → skip.
4. **TLS handling**: apply `proxmox.WithInsecure(true)` when
   `PVE_E2E_INSECURE=1`, or `proxmox.WithCACert(path)` when
   `PVE_E2E_CA_CERT` is set.
5. **Smoke check**: perform a cheap authenticated request (e.g.
   `client.Version()` or `client.Cluster().Status()`) to fail fast with a
   clear message if the connection/credentials are wrong.
6. **Cleanup**: a `Close()` on the client and a `t.Cleanup` hook.

### 3.1 Proposed layout

```
tests/e2e/
├── main_test.go          // TestMain: env parsing, client build, smoke check, skip logic
├── client.go             // NewE2EClient(t) helper + E2EConfig struct
├── helpers.go            // shared assertions, name generation, cleanup registry
├── storage/
│   └── storage_test.go   // storage module lifecycle tests
├── cluster/
│   └── cluster_test.go   // cluster module tests
├── pools/
│   └── pools_test.go     // pools module tests
├── nodes/
│   └── nodes_test.go     // nodes module tests
├── access/
│   └── access_test.go    // access module tests
└── version/
    └── version_test.go   // version module tests
```

> Note: the exact file names are illustrative. The key point is that each
> module gets its own subfolder under `tests/e2e/`, and all of them share the
> connection helper from the parent package.

### 3.2 Shared helpers (in `helpers.go`)
- `uniqueName(prefix string) string` — returns a name like
  `e2e-<unixnano>` so parallel/rerun-safe.
- `requireNoError(t, err)` / `requireError(t, err)` — thin wrappers.
- `assertStorageEqual(t, got, want)` — field-by-field comparison helpers per
  module type.
- A **cleanup registry**: each test registers created resources so they are
  deleted even if the test fails mid-way (`t.Cleanup`).

---

## 4. Common Test Lifecycle Pattern

Every module that supports full CRUD follows the same lifecycle. This is the
core pattern the user requested:

```
1. list        → baseline: capture existing resources (should not contain our name)
2. get         → (optional) confirm our resource does not exist yet → expect IsNotFound
3. create      → create a uniquely-named resource
4. get         → verify the create: resource exists, fields match what we sent
5. update      → mutate one or more fields
6. get         → verify the update: fields reflect the new values
7. delete      → remove the resource
8. list        → verify the delete: resource no longer present
```

### 4.1 Assertion strategy
- **List**: assert the returned slice contains/does-not-contain a given ID.
- **Get**: assert the returned struct's fields match expected values.
- **Create**: assert no error; then `get` to confirm.
- **Update**: assert no error; then `get` to confirm changed fields and that
  unchanged fields are preserved.
- **Delete**: assert no error; then `get` → expect `IsNotFound`, and `list` →
  expect the ID absent.

---

## 5. Module Test Plans

Each subsection below is the plan for the module's subfolder. Modules that
only support read operations (e.g. `version`, `cluster` status) use a reduced
subset of the lifecycle.

### 5.1 `storage/` — full CRUD (primary example)

Storage is the reference module because it supports the complete lifecycle.

| Step | Call | Verify |
|------|------|--------|
| 1. list | `Storage().List(ctx, "")` | no error; returns `[]Storage` |
| 2. get (absent) | `Storage().Get(ctx, name)` | `IsNotFound` |
| 3. create | `Storage().Create(ctx, &Options{ID, Type:"dir", Content:["iso","vztmpl"], Path})` | no error |
| 4. get | `Storage().Get(ctx, name)` | `Storage.Storage == name`, `Type == "dir"`, `Content` contains `iso`, `Path` matches |
| 5. update | `Storage().Update(ctx, name, &Options{Comment: ptr("e2e"), MaxFiles: ptr(3)})` | no error |
| 6. get | `Storage().Get(ctx, name)` | `Comment == "e2e"`, `MaxFiles == 3`, `Type`/`Path` unchanged |
| 7. delete | `Storage().Delete(ctx, name)` | no error |
| 8. list | `Storage().List(ctx, "")` | `name` absent |

Additional storage-specific cases:
- `List` with a `type` filter (e.g. `"dir"`) returns only that type.
- `Create` with missing `ID`/`Type` returns a validation error (client-side).
- `Delete` of a storage still in use returns an API error (documented, may be
  skipped if the cluster has no such storage).

### 5.2 `pools/` — full CRUD

| Step | Call | Verify |
|------|------|--------|
| 1. list | `Pools().List(ctx)` | no error |
| 2. get (absent) | `Pools().Get(ctx, name)` | `IsNotFound` |
| 3. create | `Pools().Create(ctx, name, &Options{Comment: ptr("e2e")})` | no error |
| 4. get | `Pools().Get(ctx, name)` | `PoolID == name`, `Comment` matches |
| 5. update | `Pools().Update(ctx, name, &Options{Comment: ptr("e2e")})` | no error |
| 6. get | `Pools().Get(ctx, name)` | `Comment == "e2e"` |
| 7. delete | `Pools().Delete(ctx, name)` | no error |
| 8. list | `Pools().List(ctx)` | `name` absent |

### 5.3 `cluster/` — status/resources read-only; HA groups full CRUD

Cluster status and resources are read-only; no create/update/delete.

| Step | Call | Verify |
|------|------|--------|
| 1. status | `Cluster().Status(ctx)` | no error; returns cluster status incl. `Name` |
| 2. resources | `Cluster().Resources(ctx)` | no error; returns `[]Resource` |
| 3. resources (type filter) | `Cluster().Resources(ctx, "vm")` | no error; all entries match type |

`Cluster().HA().Groups()` supports the full CRUD lifecycle from §4, bound to
the node named by `PVE_E2E_NODE` (the test is skipped when unset):

| Step | Call | Verify |
|------|------|--------|
| 1. list | `HA().Groups().List(ctx)` | no error |
| 2. get (absent) | `HA().Groups().Get(ctx, name)` | error |
| 3. create | `HA().Groups().Create(ctx, &GroupOptions{ID, Nodes: &node, Comment})` | no error |
| 4. get | `HA().Groups().Get(ctx, name)` | `Group == name`, `Nodes == node`, `Comment` matches |
| 5. update | `HA().Groups().Update(ctx, name, &GroupOptions{Comment, Restricted, Nofailback})` | no error |
| 6. get | `HA().Groups().Get(ctx, name)` | `Comment`/`Restricted`/`Nofailback` updated, `Nodes` unchanged |
| 7. delete | `HA().Groups().Delete(ctx, name)` | no error |
| 8. list | `HA().Groups().List(ctx)` | `name` absent |

`Cluster().HA().Rules()` also supports the full CRUD lifecycle, using a
node-affinity rule bound to `PVE_E2E_NODE` and a synthetic (non-existent)
`vm:<id>` resource — HA rules are plain configuration entries, so Proxmox
does not require the referenced guest to exist:

| Step | Call | Verify |
|------|------|--------|
| 1. list | `HA().Rules().List(ctx, "", "")` | no error |
| 2. get (absent) | `HA().Rules().Get(ctx, name)` | error |
| 3. create | `HA().Rules().Create(ctx, &RuleOptions{ID, Type: NodeAffinity, Resources, Nodes: &node, Comment})` | no error |
| 4. get | `HA().Rules().Get(ctx, name)` | `Rule == name`, `Type`/`Resources`/`Nodes`/`Comment` match |
| 5. update | `HA().Rules().Update(ctx, name, &RuleOptions{Type: NodeAffinity, Comment, Disable})` (Type must be resent) | no error |
| 6. get | `HA().Rules().Get(ctx, name)` | `Comment`/`Disable` updated, `Resources` unchanged |
| 7. delete | `HA().Rules().Delete(ctx, name)` | no error |
| 8. list | `HA().Rules().List(ctx, "", "")` | `name` absent |
| type filter | `HA().Rules().List(ctx, "node-affinity", "")` vs `"resource-affinity"` | our rule present only under its own type |

### 5.4 `nodes/` — read-only (list/get)

Node listing is read-only.

| Step | Call | Verify |
|------|------|--------|
| 1. list | `Nodes().List(ctx)` | no error; returns `[]Node` |
| 2. get | `Nodes().Get(ctx, node)` | no error; `Node == node` |
| 3. get (absent) | `Nodes().Get(ctx, "does-not-exist")` | `IsNotFound` |

### 5.5 `access/` — read-only (users/roles/groups)

Access is read-only in this plan (creating users is out of scope for v1).

| Step | Call | Verify |
|------|------|--------|
| 1. users | `Access().Users(ctx)` | no error; returns `[]User` |
| 2. roles | `Access().Roles(ctx)` | no error; returns `[]Role` |
| 3. groups | `Access().Groups(ctx)` | no error; returns `[]Group` |

### 5.6 `version/` — read-only

| Step | Call | Verify |
|------|------|--------|
| 1. version | `client.Version(ctx)` | no error; `Version`/`Release` non-empty |

---

## 6. Authentication Tests

A dedicated test file (e.g. `tests/e2e/auth_test.go`) verifies both auth
paths against the live cluster:

| Case | Setup | Verify |
|------|-------|--------|
| Token auth | `PVE_E2E_TOKEN_ID`/`SECRET` set | a `Version()` call succeeds |
| Password auth | `PVE_E2E_USERNAME`/`PASSWORD` set | a `Version()` call succeeds |
| Bad token | wrong secret | returns an API error (401) |
| Bad password | wrong password | returns an API error (401) |
| Session renewal | (optional) force ticket expiry | client transparently renews and continues |

---

## 7. Error Handling Tests

| Case | Call | Verify |
|------|------|--------|
| Not found | `Get` on a non-existent resource | `proxmox.IsNotFound(err) == true` |
| Bad request | `Create` with invalid params | returns `*proxmox.APIError` with `StatusCode >= 400` |
| Rate limited | (optional, hard to trigger) | `proxmox.IsRateLimited(err)` on a 429 |

---

## 8. Cleanup & Isolation

- Every created resource uses a unique name from `uniqueName()`.
- Each test registers a `t.Cleanup` that deletes the resource if it still
  exists, so a failed test does not leak resources.
- Cleanup on failure is **on by default** (`PVE_E2E_CLEANUP_ON_FAILURE=true`).
  Set it to `false` to keep resources around after a failure for manual
  inspection/debugging.
- The suite never touches resources it did not create (it only asserts
  absence/presence of its own names).
- A final `list` in each test confirms cleanup.

---

## 9. Running the Suite

Documented in `Makefile` (new targets) and this doc:

```sh
# token auth
PVE_E2E_URL=https://pve:8006/api2/json \
PVE_E2E_TOKEN_ID='root@pam!e2e' \
PVE_E2E_TOKEN_SECRET='...' \
PVE_E2E_INSECURE=1 \
go test -tags=e2e ./tests/e2e/... -v

# password auth
PVE_E2E_URL=https://pve:8006/api2/json \
PVE_E2E_USERNAME='root@pam' \
PVE_E2E_PASSWORD='...' \
go test -tags=e2e ./tests/e2e/... -v
```

### 9.1 Makefile targets (planned)
- `make e2e` — runs the full e2e suite (requires env vars).
- `make e2e-storage` — runs only the storage module tests.

---

## 10. Open Questions / Decisions

- **Module coverage order**: implement `storage` first (reference CRUD), then
  `pools`, then read-only modules. Confirm this ordering.
- **`access` scope**: should v1 include creating a test user/role, or stay
  read-only? (Currently planned read-only.)
- **`nodes` scope**: v1 is read-only list/get. VM/QEMU lifecycle is a future
  milestone.
- **Parallelism**: module subfolders run sequentially by default. Set
  `PVE_E2E_PARALLEL=true` to run them in parallel (`t.Parallel()`). Unique
  names make it safe, but cluster-side task serialization may argue for
  sequential runs.
- **Cleanup on failure**: `t.Cleanup` registry is used, and cleanup on
  failure is on by default (`PVE_E2E_CLEANUP_ON_FAILURE=true`). Set it to
  `false` to retain resources for debugging.

# Node-Affinity Load Balancing — Design

> Route `/nodes/{node}/...` requests straight at the node that owns them, and fall
> back to the ordinary load-balancing algorithm when that node is not reachable.

Status: **Phases 1–3 implemented** (§11): `nodeFromPath`, the route context,
`NodeBalancer` with delegate fallback, the static-map/template/`WithNodeResolver`/
pool-matching endpoint sources, passive health, first-attempt-only direct routing,
`OnRoute`, the `basePath` fix from §9, and `WithSafePOSTFailover`'s
provably-unsent-only POST retry. It applies automatically to any configured balancer
(§13 item 1) — `WithNodeAffinity` is now for configuring it, not turning it on. Phase 4
(guest→node affinity, websocket pinning) remains deferred; see §1's non-goals.
Section 13's questions are decided except item 5, for the same reason. The
`WithCACert` multi-CA fix from §9 ships separately, in `feat/multi-ca-cert`.

---

## 1. Motivation

Every node in a Proxmox VE cluster exposes the *full* API surface. A request for
`/nodes/pve2/qemu/100/status/current` sent to `pve1` is accepted by `pve1`'s
`pveproxy`, which recognises the node name in the path, opens a second HTTPS
connection to `pve2:8006` and relays the request and response. The client never sees
the extra hop.

That hop is free only when nothing goes wrong. In practice it costs us:

- **Latency and connection churn.** Every node-scoped call becomes two TLS
  connections instead of one. The relayed connection is not pooled with the same
  lifetime as ours, so bursts of per-node calls (inventory walks, task polling) pay
  handshake cost repeatedly.
- **Body streaming.** `POST /nodes/{node}/storage/{storage}/upload` relays the whole
  ISO/template body through the front node. `pveproxy` is already the reason
  `Client.Upload` has to compute an exact `Content-Length` by hand (it rejects
  `Transfer-Encoding: chunked` with `501`, see `client.go`); relaying doubles the
  traffic and doubles the exposure to that code path. Same for
  `storage/{storage}/download-url` progress and large `tasks/{upid}/log` reads.
- **Error attribution.** When the relay fails, `pveproxy` returns `595 errors during
  connection` or a `500` whose message describes the *relay's* problem, not the
  target node's. The caller cannot distinguish "pve2 is down" from "pve1 cannot
  reach pve2". Talking to `pve2` directly turns that into a plain transport error we
  can classify.
- **Blast radius.** With `WithRoundRobin` the front node is picked arbitrarily, so a
  single sick node degrades *every* node-scoped call that happens to land on it,
  not just calls for that node.

Direct routing removes the hop for the one class of request where we statically know
the correct destination: any path of the form `/nodes/{node}/...`. Everything else
(`/cluster/*`, `/storage/*`, `/pools/*`, `/access/*`, `/version`) is genuinely
cluster-wide and keeps using the configured algorithm.

### Non-goals

- Replacing the existing balancers. Node affinity is a *decorator* over
  round-robin / weighted round-robin / SRV, not a fourth algorithm.
- Guest-level affinity (resolving `vmid` → node via `/cluster/resources`). That needs
  an API call and a cache with its own invalidation story, and the node name is
  already in the path for every endpoint we care about. Deferred; see §13.
- Websocket endpoints (`vncwebsocket`, `termproxy`, `spiceproxy`). This client does
  not implement them yet. When it does they *must* be pinned — the ticket those
  endpoints return is only valid on the issuing node — so the mechanism below is
  designed to be reusable there.
- Cross-node request hedging. resty ships `hedging.go`; combining it with affinity is
  out of scope.

---

## 2. What resty gives us, and what it does not

The whole design is shaped by one constraint, so it is worth stating precisely.

`resty.LoadBalancer` (`load_balancer.go`) is:

```go
type LoadBalancer interface {
	NextWithContext(ctx context.Context) (string, error)
	Feedback(*RequestFeedback)
	Close() error
}

type RequestFeedback struct {
	BaseURL string
	Success bool
	Attempt int
}
```

Four facts follow from reading resty's call sites:

1. **`NextWithContext` sees the context, not the request.** It is called from
   `parseRequestURL` (`middleware.go:137`) with `r.Context()`. The request path is
   *not* passed. **Therefore the node hint has to travel in the request context** —
   there is no other seam. This is the pivot of the design.

2. **It is called once per attempt, not once per request.** `Request.Execute`
   (`request.go:1509`) resets `r.URL = url` and re-runs the middleware chain on every
   retry, so each attempt re-enters the balancer. Intra-request failover is therefore
   possible *inside* `NextWithContext` and nowhere else.

3. **`Feedback` is called once, after the last attempt** (`request.go:1612`), with the
   base URL of that last attempt. resty marks `success = false` only for
   `ECONNREFUSED` (non-timeout) and for `5xx` other than `501`
   (`request.go:1855-1881`). So feedback is a coarse, end-of-request signal — good
   enough to drive a circuit breaker, useless for failing over mid-request.

4. **Retries only happen for idempotent methods.** `idempotentMethods`
   (`request.go:1895`) is `DELETE, GET, HEAD, OPTIONS, PUT, QUERY, TRACE`. A `POST`
   gets exactly one attempt unless `IsRetryAllowNonIdempotent` is set. Since Proxmox
   uses `POST` for almost every mutation, **a pinned `POST` that hits a dead node
   fails outright**. §7.2 addresses this.

Two smaller facts we rely on:

- `Client.Close` closes the balancer (`resty/client.go:2435`), so a balancer with a
  background refresh goroutine is cleaned up by the existing `proxmox.Client.Close`.
- `RoundRobin`/`WeightedRoundRobin` expose `Refresh` but **no getter for their base
  URL list**. If we want to derive node endpoints from the pool (§6.3) we must keep
  our own copy of the URLs at option-application time.

---

## 3. Architecture

```
 proxmox.Client.send(ctx, method, path, out, req)
        │
        │ 1. node, ok := nodeFromPath(path)          //  "/nodes/pve2/..." -> "pve2"
        │ 2. req.SetContext(withRoute(ctx, &route{node: node}))
        ▼
 resty.Request.Execute  ── per attempt ──▶ parseRequestURL
                                                │
                                                ▼
                                    NodeBalancer.NextWithContext(ctx)
                                                │
                    ┌───────────────────────────┼───────────────────────────┐
                    │                           │                           │
            route in ctx?                 endpoint known?              endpoint healthy?
              no │                          no │                         no │
                 └──────────────┬──────────────┴──────────────┬────────────┘
                                ▼                             ▼
                        delegate.NextWithContext(ctx)   (same — fall back)

                              yes / yes / yes  ──▶  return endpoint.BaseURL (direct)
        │
        ▼
 resty.Request ── Feedback(BaseURL, Success) ──▶ NodeBalancer.health.observe(...)
                                                  └─ also forwarded to delegate
```

Three collaborators:

| Component | Responsibility |
|---|---|
| `nodeFromPath` + route context | Turn a request path into a routing hint and carry it to the balancer. |
| `registry` | Answer "which base URL reaches node `pve2`?" |
| `health` | Answer "is that base URL usable right now?" and absorb `Feedback`. |

`NodeBalancer` wires the three together and implements `resty.LoadBalancer`. It
**wraps a delegate** `resty.LoadBalancer` (the user's round-robin / WRR / SRV, or a
single-URL balancer synthesised from `BaseURL`) which it consults for every request it
does not pin.

### 3.1 File layout

```
loadbalancer.go        // NodeBalancer, route context, nodeFromPath
loadbalancer_health.go // health registry (failure counting, recovery ticker)
loadbalancer_disco.go  // endpoint sources: static map, template, pool matching
options.go             // WithNodeAffinity + NodeAffinityOption (existing file)
loadbalancer_test.go
```

All of it lives in the **root `proxmox` package**, next to the `Option` values that
configure it and the `send` that produces the routing hint. Nothing here needs a
`*proxmox.Client`, so the placement is for cohesion rather than to dodge an import
cycle — no endpoint source calls the API (§6).

---

## 4. The routing hint

### 4.1 Extraction

```go
// nodeFromPath returns the node name from a node-scoped API path, i.e. the
// {node} in /nodes/{node}/... . It reports false for /nodes and /nodes/ (the
// cluster-wide node list) and for every non-node path.
func nodeFromPath(p string) (string, bool)
```

Rules:

- Match on the path **as passed to `send`**, before `Client.path` prepends `basePath`.
  `send` already receives `/nodes/pve2/status`, so there is no prefix to strip.
- `/nodes` and `/nodes/` → no hint. `GET /nodes` is the cluster-wide node list and
  every node answers it.
- `/nodes/{node}` with nothing after it → hint. `GET /nodes/pve2` is the node's own
  index and is cheapest at the node.
- Nothing else is special-cased. Every `/nodes/{node}/...` endpoint is either served
  by that node or relayed to it by `pveproxy`; pinning is never *wrong*, only
  sometimes not *better*.
- The node segment is used verbatim. Proxmox node names are DNS labels
  (`[a-zA-Z0-9-]`), and this client builds paths by plain concatenation, so no
  unescaping is needed. A segment containing `%` or `/` yields no hint rather than a
  guess.

### 4.2 Carrying it

```go
// route is the per-request routing state. It is created by send, carried in
// the request context, and mutated by NextWithContext on each attempt.
type route struct {
	node     string
	attempts int      // incremented by NextWithContext
	pinned   []string // base URLs already tried directly
}

func withRoute(ctx context.Context, r *route) context.Context
func routeFrom(ctx context.Context) *route
```

A **pointer** in the context, deliberately. The context value cannot be replaced
between retry attempts (resty holds one `*Request`), so the only way for attempt *n*
to know what attempt *n-1* did is a mutable cell. This is what makes §7.2's
intra-request failover possible without any resty hook.

`send` is the single place that sets it, which means it covers `do`, `doValues` and
`Upload` — every public verb — for free. Requests issued outside `send`
(`Client.ticket`) simply carry no route and are delegated, which is correct: the
ticket endpoint is cluster-wide.

`routeFrom` returning `nil` is the normal, expected case, not an error.

---

## 5. Routing decision

```go
func (b *NodeBalancer) NextWithContext(ctx context.Context) (string, error) {
	r := routeFrom(ctx)
	if r == nil || r.node == "" {
		return b.delegate.NextWithContext(ctx)      // not node-scoped
	}

	r.attempts++
	if r.attempts > maxDirectAttempts {             // §7.2 — const 1
		return b.delegate.NextWithContext(ctx)
	}

	ep, ok := b.registry.lookup(r.node)             // §6
	if !ok {
		b.observe(r.node, routeUnknown)
		return b.delegate.NextWithContext(ctx)      // unmapped node
	}
	if !b.health.usable(ep.BaseURL) {               // §7
		b.observe(r.node, routeUnhealthy)
		return b.delegate.NextWithContext(ctx)      // known-bad endpoint
	}

	r.pinned = append(r.pinned, ep.BaseURL)
	b.observe(r.node, routeDirect)
	return ep.BaseURL, nil
}
```

Notes:

- **Falling back is the only behaviour, and it is never silent.** The balancer never
  fails a request because a node could not be reached directly: a relayed answer is
  still a correct answer, and refusing to relay would make enabling affinity a
  semantic change rather than a routing optimisation. Every miss is reported through
  the `OnRoute` hook (§8), so an operator can see how often affinity is not paying
  off. A caller that genuinely needs to know whether one node is reachable should ask
  that directly — `Nodes(n).Version()` against a pinned endpoint — rather than infer
  it from an error the balancer manufactured.
- **Falling back also preserves error *shape*.** When the target node is down, the
  relay path returns a structured Proxmox error (`595`/`500` with an `errors` map)
  that `newAPIError` turns into `*APIError`; the direct path returns a bare transport
  error. Callers that branch on `IsNotFound`/`IsUnexpected` keep working because we
  fall back once the node is known-bad, so only the *first* failure of a newly-dead
  node surfaces as a transport error. This asymmetry must be documented on the option.

---

## 6. Node → endpoint registry

The hard part is not the routing, it is knowing that node `pve2` is reachable at
`https://10.0.0.2:8006`. Three sources, consulted in this precedence order; the first
that yields an endpoint wins.

**No source calls the Proxmox API.** Deriving the map from `GET /cluster/status` was
considered and rejected: `NodeStatus.IP` is the corosync link address, not necessarily
an address `pveproxy` listens on, so on any cluster with a dedicated corosync network
— a recommended topology — it is simply the wrong address. A source that is wrong on
well-built clusters is worse than no source. Callers who do have a trustworthy dynamic
mapping supply it through `NodeResolver` (§6.4) instead.

### 6.1 Static map — `WithNodeEndpoints`

```go
WithNodeAffinity(
    WithNodeEndpoints(map[string]string{
        "pve1": "https://10.0.0.1:8006",
        "pve2": "https://10.0.0.2:8006",
    }),
)
```

Explicit, no I/O, no surprises. The right answer for Kubernetes/CCM-style deployments
where the node list comes from config anyway. Recommended for production.

### 6.2 Template — `WithNodeEndpointTemplate`

```go
WithNodeEndpointTemplate("https://{node}.pve.example.com:8006")
```

`{node}` is substituted with the node name. One line for the common case where node
names resolve in DNS. Cheap, but unlike the other two sources it cannot tell a real
node from a typo: it produces an endpoint for every name it is handed. With health
being passive (§7), a bogus name is only discovered when a request to it fails, so
this source trades a little safety for its brevity.

### 6.3 Pool name matching — default, zero-config

Derive the mapping from the base URLs the user *already* gave the balancer:
`https://pve1.example.com:8006` → node `pve1`. Matching is case-insensitive against,
in order, the full host and the first DNS label. A host that is a bare IP address
never matches (there is nothing to match on).

This is the default because it costs nothing, makes no requests, and is correct for
the overwhelmingly common `WithRoundRobin("https://pve1...", "https://pve2...")`
setup.

**It requires a change to how the pool is recorded.** `resty.RoundRobin` and
`WeightedRoundRobin` have no accessor for their base URLs, so `WithRoundRobin`,
`WithWeightedRoundRobin` and `WithURL` must additionally stash the URLs on
`ClientConfig` (a new unexported `lbURLs []string`) for the registry to read.
`WithLoadBalancer` and `WithSRVWeightedRoundRobin` cannot contribute here — a custom
balancer is opaque and SRV targets are resolved inside resty — so with those, pool
matching yields nothing and one of the other two sources is required.

### 6.4 Registry interface

Sources are normalised behind one interface so a caller can supply their own (e.g.
one backed by a Kubernetes informer):

```go
// NodeEndpoint binds a Proxmox node name to the base URL that reaches its API.
type NodeEndpoint struct {
	Node    string
	BaseURL string // scheme://host[:port], no path — LB endpoints carry no path
}

// NodeResolver resolves the cluster's node endpoints. Implementations must be
// safe for concurrent use.
type NodeResolver interface {
	Nodes(ctx context.Context) ([]NodeEndpoint, error)
}

WithNodeResolver(r NodeResolver, refresh time.Duration) NodeAffinityOption
```

`BaseURL` must be path-free: §9 of this document and `Client.path` both assume LB
endpoints carry no path, with `basePath` supplying it. The registry normalises with
the same `url.Path = ""` treatment resty's `extractBaseURL` applies.

The three built-in sources are static: they are resolved once, when the balancer is
built. `WithNodeResolver` is the only one that refreshes, on its own goroutine started
by `New` and stopped by `NodeBalancer.Close` (which resty calls from `Client.Close`).
A refresh that fails **keeps the previous map** and logs, matching how resty's own SRV
balancer handles a failed lookup (`load_balancer.go`, `swrr.ticker`). The resolver must
not route its own traffic through this client's balancer; if a caller's implementation
does call the Proxmox API, it owns that client and the re-entrancy question that comes
with it. The registry lock is never held across a resolver call — the goroutine
fetches first, then swaps the map under the lock.

---

## 7. Health

Health is **passive only**: an endpoint's usability is inferred from the outcomes of
the requests the client was going to make anyway. The balancer never probes, never
polls and never asks the cluster for an opinion — it has no traffic of its own.

Active probing (`GET /version` per node on a ticker) and source-reported state (a
resolver declaring a node offline) were both considered and dropped. Probing spends a
request per node per interval to learn something the next real request would reveal
anyway, and it can disagree with reality between ticks — a node that answers
`/version` can still fail the call you care about. Statistics from real traffic have
neither problem, and they are what resty's own `WeightedRoundRobin` uses.

A node endpoint is **usable** when it is not in the circuit breaker's open state.

### 7.1 Circuit breaker from `Feedback`

Mirrors `WeightedRoundRobin`'s semantics deliberately, so the behaviour is familiar:

- `Feedback{Success:false}` increments a per-base-URL failure counter.
- At `MaxFailures` (default 5) the endpoint goes **open** and stops being used for
  direct routing.
- A recovery ticker (default 120s, matching resty's WRR default) closes every open
  endpoint and zeroes its counter. There is no half-open probe: the first real request
  routed after recovery *is* the probe, and if it fails the endpoint reopens after
  `MaxFailures` again. This is the passive-only rule applied to recovery as well as to
  detection.
- A successful feedback resets the counter to zero.

`Feedback` is **also forwarded to the delegate**, unconditionally. The delegate may be
a WRR with its own health state over the same base URLs; swallowing the signal would
leave it blind.

Because feedback is keyed by base URL and both layers key on the same strings, a node
that is dead is dropped by both the affinity layer and the WRR pool. That is the
desired behaviour, and it is why health state is keyed by **base URL, not node name**.

The cost of going passive-only is worth stating: an endpoint that is never routed to
is never evaluated, so a node that dies while idle is not noticed until the first
request for it — which fails (§7.2). `MaxFailures` requests must fail before the
breaker opens, so a node that dies under low traffic for that node can absorb several
failures over a long window. Lowering `MaxFailures` to 1 via `WithNodeMaxFailures`
trades sensitivity to a single blip for faster shedding; callers who want that should
set it explicitly.

### 7.2 Intra-request failover, and the POST problem

```go
// maxDirectAttempts is the attempt number after which a pinned request stops
// going direct and falls back to the delegate balancer.
const maxDirectAttempts = 1
```

Attempt 1 goes direct; every retry goes through the delegate. That converts "pinned
node just died" into "one wasted attempt" for idempotent methods — resty re-runs the
middleware chain per attempt (§2 fact 2), so attempt 2 re-enters `NextWithContext`,
sees `r.attempts == 2`, and delegates.

This is a **constant, not an option**. Any value above 1 means "retry the same dead
node before trying anyone else", which is only ever useful when the delegate pool has
a single member — and in that case falling back and retrying direct are the same
request. A knob whose only correct setting is its default is a knob that invites
misconfiguration, so it stays unexported.

For **`POST`** there is no attempt 2 (§2 fact 4). Options considered:

| Option | Verdict |
|---|---|
| Set `IsRetryAllowNonIdempotent` on POSTs | **Rejected.** Proxmox POSTs create VMs, start migrations and fire tasks. A blind retry could double-execute. |
| Retry POST only when the transport error proves the request was never sent (`ECONNREFUSED`, DNS failure, TLS handshake failure) | **Accepted for phase 3** (§11), implemented as `WithSafePOSTFailover`. Safe, because a refused connection cannot have reached the API. Uses `Request.SetRetryConditions` (not `AddRetryConditions`) to *replace* the request's retry conditions rather than add to them — `Client.retryCondition`'s own `err != nil` branch is not safe to reuse for a non-idempotent method — plus `Request.SetRetryAllowNonIdempotent`, applied only to a POST on a node-scoped path when the option is set. Detection is `*net.OpError` with `Op == "dial"` (covers `ECONNREFUSED`, DNS failures and "no route to host" alike — Go's transport wraps all of them there) or `*tls.CertificateVerificationError` (every handshake-time certificate failure); it does *not* apply once bytes are on the wire. |
| Accept the single failure and rely on the circuit breaker | **Accepted for phase 1.** The first POST after a node dies fails with a transport error; subsequent ones fall back. Bounded, explainable, and no correctness risk. |

Phase 1 ships the third row and documents it. This is the single sharpest edge of the
feature and should be stated plainly in the option's doc comment.

---

## 8. Public API

```go
// WithNodeAffinity routes /nodes/{node}/... requests directly at the node
// that owns them, falling back to the configured load-balancing algorithm
// when the node's endpoint is unknown or unreachable.
//
// Any configured balancer (WithRoundRobin, WithWeightedRoundRobin,
// WithSRVWeightedRoundRobin, WithLoadBalancer) gets this automatically,
// using zero-config pool matching as its endpoint source (§13 item 1) —
// calling WithNodeAffinity is only needed to configure an explicit source,
// tune health, install WithOnRoute, or (WithURL alone, no balancer) to
// enable the feature at all. Apply it after the balancer option.
func WithNodeAffinity(opts ...NodeAffinityOption) Option

type NodeAffinityOption func(*nodeAffinityConfig)

// Endpoint sources (§6), highest precedence first. Pool matching is the
// default and needs no option.
func WithNodeEndpoints(m map[string]string) NodeAffinityOption
func WithNodeEndpointTemplate(tmpl string) NodeAffinityOption
func WithNodeResolver(r NodeResolver, refresh time.Duration) NodeAffinityOption

// Health (§7) — passive, driven by resty's Feedback.
func WithNodeMaxFailures(n int) NodeAffinityOption
func WithNodeRecovery(d time.Duration) NodeAffinityOption

// Behaviour. Direct routing is tried on the first attempt only; see §7.2 for
// why that is a constant rather than an option. WithSafePOSTFailover is the
// one exception, phase 3 (§11): opt-in retry for a POST against a pinned
// node, but only when the transport error proves it was never sent.
func WithOnRoute(fn func(node, baseURL string, d RouteDecision)) NodeAffinityOption
func WithSafePOSTFailover() NodeAffinityOption

type RouteDecision int
const (
	RouteDirect RouteDecision = iota
	RouteUnknown
	RouteUnhealthy
	RouteNotNodeScoped
)
```

Affinity never changes what a call returns, only which host serves it: every miss
falls back (§5). That is what makes `WithNodeAffinity` safe to add to an existing
client and what lets §10's E2E suite assert the same results with it on and off.

### Usage

```go
// Zero-config: pool names already carry the node names (§6.3), and node
// affinity applies automatically — no WithNodeAffinity call needed.
c, err := proxmox.New(proxmox.ClientConfig{},
	proxmox.WithRoundRobin(
		"https://pve1.example.com:8006",
		"https://pve2.example.com:8006",
		"https://pve3.example.com:8006",
	),
	proxmox.WithTokenAuth("root@pam!ci", secret),
)

// Behind a VIP, with an explicit node map. WithURL alone installs no
// balancer, so WithNodeAffinity is what enables the feature here; the VIP
// stays the fallback.
c, err := proxmox.New(proxmox.ClientConfig{},
	proxmox.WithURL("https://pve-vip.example.com:8006/api2/json"),
	proxmox.WithNodeAffinity(proxmox.WithNodeEndpoints(map[string]string{
		"pve1": "https://10.0.0.1:8006",
		"pve2": "https://10.0.0.2:8006",
	})),
)

// A non-standard API prefix: the endpoints stay bare host:port, WithBasePath
// supplies the path for all of them. Still automatic — WithNodeAffinity here
// is only for the WithOnRoute hook.
c, err := proxmox.New(proxmox.ClientConfig{},
	proxmox.WithRoundRobin("https://pve1.example.com:8006", "https://pve2.example.com:8006"),
	proxmox.WithBasePath("/pve/api2/json"),
	proxmox.WithNodeAffinity(proxmox.WithOnRoute(logRouteDecision)),
)
```

### 8.1 Where the API path comes from

`WithURL` exists for back-compatibility and accepts a URL **with or without** a path.
Without a balancer it is handed straight to resty as the base URL, so whatever path it
carries is applied by resty and `Client.path` stays a no-op.

Under a balancer that stops being true: an endpoint is `scheme://host[:port]` and
nothing more — resty's own `extractBaseURL` strips the path off every base URL it is
given (`load_balancer.go`), and node endpoints (§6.4) are bare hosts by construction.
The API prefix therefore has to come from `basePath`, which `Client.path` prepends to
every request.

**`WithNodeAffinity` always installs a balancer**, so it always puts the client in
that mode, even for a caller who only passed `WithURL`. The prefix is resolved as:

| Source | Sets `basePath` to |
|---|---|
| `WithBasePath("/pve/api2/json")` | that value, trimmed of a trailing `/` |
| `WithURL("https://vip:8006/api2/json")` | the URL's path, `/api2/json` |
| `WithURL("https://vip:8006")` | nothing — the URL carries no path |
| neither, or both left empty | `defaultBasePath`, `/api2/json`, applied by `New` |

`WithBasePath` and a path-carrying `WithURL` are both explicit statements of intent,
so between those two the documented **last-one-wins** rule of `Option` applies
unchanged — `WithURL(".../api2/json")` after `WithBasePath("/custom")` really does
mean `/api2/json`. What must *not* happen is a balancer option silently overwriting
either of them; see §9.

So `WithURL("https://vip:8006/api2/json")` plus `WithNodeAffinity()` keeps working
byte-for-byte: the path moves from resty's base URL into `basePath` and is re-applied
by `Client.path`. The synthesised single-URL delegate is built from `BaseURL` with its
path stripped, matching what `extractBaseURL` would have done anyway.

---

## 9. Interactions and caveats

**TLS.** Two separate checks, and only one of them is a real obstacle.

*Chain of trust — solved by `WithCACert`.* The cluster CA lives at
`/etc/pve/pve-root-ca.pem`, inside the replicated `/etc/pve`, so it is **identical on
every node**, and each node's `/etc/pve/local/pve-ssl.pem` is signed by it. One
`WithCACert` pointed at that CA therefore validates the chain for every node in the
cluster. Direct routing needs no per-node trust configuration.

*Name matching — the actual constraint.* Verification also requires the host in the
endpoint URL to appear in that node's certificate SANs. PVE's generated `pve-ssl.pem`
carries the node name, its FQDN and its local IP addresses, so both
`https://pve2.example.com:8006` and `https://10.0.0.2:8006` normally verify — but SAN
contents depend on how the cluster was set up and on any later `pvecm updatecerts`, so
the endpoint map (§6.1) must use names the certificates actually carry. Confirm once
per cluster with:

```
openssl x509 -in /etc/pve/local/pve-ssl.pem -noout -text | grep -A1 'Subject Alternative Name'
```

*The case that genuinely breaks.* A cluster fronted by a VIP or ingress with an
ACME/public certificate, whose nodes still present PVE-CA-signed certificates: the two
now need **different** trust anchors. And `WithCACert` cannot currently express that —
it takes one path, and resty's `handleCAs` builds a **fresh** `x509.NewCertPool()`
rather than starting from `x509.SystemCertPool()`, so setting it *replaces* the system
trust store instead of adding to it. Trusting the PVE CA today means un-trusting the
public CA that the VIP's certificate chains to.

`resty.Client.SetRootCertificates` is already variadic, so the fix is small and should
ship with this feature: make `WithCACert` accept multiple paths (or add
`WithCACerts(...string)`), and document that supplying any CA replaces the system pool
— so a mixed deployment must list both the PVE CA and the public CA. `WithInsecure(true)`
remains the escape hatch. The doc comment on `WithNodeAffinity` should point here.

**Authentication.** Both schemes survive the redirect. API tokens are cluster-wide
(`/etc/pve` is replicated). Ticket auth also works cluster-wide: the ticket is signed
with the cluster-shared authkey, which is exactly why `pveproxy` can relay an
authenticated request in the first place. `send` sets the `PVEAuthCookie` header
explicitly rather than relying on resty's cookie jar, so the ticket follows the
request to whichever host the balancer picks — a jar keys by host and would not.
`Client.ticket` itself is unpinned and may land on any node; that is fine.

**`basePath`.** §8.1 makes `basePath` the only source of the API prefix whenever a
balancer is installed, which is now the common case. Two existing rough edges become
load-bearing and should be fixed with this work:

- `WithRoundRobin`/`WithWeightedRoundRobin`/`WithSRVWeightedRoundRobin` set
  `c.basePath = defaultBasePath` unconditionally, so `WithBasePath("/pve/api2/json")`
  followed by `WithRoundRobin(...)` silently loses the custom prefix. They should not
  touch `basePath` at all: `New` already defaults it when a balancer is present and
  the field is empty, which is the correct single place for that. `WithNodeAffinity`
  must follow the same rule — install the balancer, leave `basePath` alone.
- `Client.ToRESTConfig` drops `basePath` (while `ClientConfig.ToRESTConfig` keeps it),
  so a client derived from it silently loses its prefix. Straightforward omission.

**`Upload`.** `Upload` builds its own `*resty.Request` and calls `send`, so it picks
up the route for free and gets the largest single win from the feature. Its
`Content-Length` handling is unaffected by which host answers.

**Tasks.** `/nodes/{node}/tasks/{upid}/status` and `/log` read files local to the node
that ran the task, and the node name is already in the path. Pinning is both faster
and more obviously correct. A UPID's node and the path's node are the same by
construction in this client.

**Migration.** `POST /nodes/{node}/qemu/{vmid}/migrate` is issued against the *source*
node, which is the `{node}` in the path, so pinning is correct. After the migration
completes, subsequent calls use the new node name and therefore a different endpoint —
no stale-affinity problem, because nothing is cached per guest.

**Concurrency.** `NextWithContext` is on the hot path of every request. The registry
is a `map[string]NodeEndpoint` swapped wholesale under an `RWMutex` on refresh
(read-mostly); health counters are per-endpoint under the same lock or a small
sharded structure. The `*route` in the context is touched by one request's attempts,
which are sequential, so it needs no lock.

**Logging.** Route decisions go through `WithOnRoute` rather than the resty logger, so
callers can turn them into metrics. A default no-op keeps the hot path free.

---

## 10. Testing

`fakeapi` is the lever here: it is already an `httptest`-mountable Proxmox, so a
"cluster" is *N* `fakeapi` servers.

**Unit.**
- `nodeFromPath` table test: `/nodes/pve1/status` → `pve1`; `/nodes` → none;
  `/nodes/` → none; `/nodes/pve1` → `pve1`; `/cluster/status` → none; `/version` →
  none; `/nodes/pve%2f1/x` → none.
- Registry precedence: static map beats template beats pool matching.
- Pool matching: `https://pve1.example.com:8006` → `pve1`; `https://10.0.0.1:8006` →
  no match; case-insensitivity.
- Health: `MaxFailures` failures opens; recovery ticker closes; success resets.
- `NextWithContext` decision matrix, driven by a stub delegate that records calls —
  one case per `RouteDecision`, asserting the delegate is consulted for every one but
  `RouteDirect`.
- `basePath` resolution (§8.1), one case per row of that table, asserting the *final
  request URL*: `WithURL(".../api2/json") + WithNodeAffinity()` must produce the same
  URL as `WithURL(".../api2/json")` alone; `WithURL("https://vip:8006")` must fall to
  `/api2/json`; `WithBasePath("/pve/api2/json") + WithRoundRobin(...)` must keep the
  custom prefix (the §9 regression); and `WithBasePath` then a path-carrying
  `WithURL` must resolve last-one-wins.

**Integration (in-process, no network).**
- Three `fakeapi` servers, each told its own node name; assert that
  `Nodes("pve2").Status(ctx)` is served by server 2 and that `Cluster().Status(ctx)`
  is served round-robin.
- Kill server 2 (`srv.Close()`); assert the first call for `pve2` fails with a
  transport error, that once the breaker opens subsequent calls are delegated, and
  that `OnRoute` reports `RouteUnhealthy`.
- Restart it, advance the recovery clock (inject the ticker), assert direct routing
  resumes.
- Retry failover: a `GET` against a dead node must go direct on attempt 1 and land on
  the delegate on attempt 2. A `POST` must not be retried.
- `Upload` against a pinned node, reusing the existing `TestUploadNoChunkedEncoding`
  harness, to prove the route survives the hand-built multipart path.

**E2E.** `docs/e2e.md`'s gated suite gets one case: with `WithNodeAffinity` against a
real cluster, `GET /nodes/{node}/status` for every node returns that node's own
`Nodes(node).Version()` — and the same suite must pass with affinity off, proving the
feature is transparent.

---

## 11. Phasing

| Phase | Content | Status |
|---|---|---|
| 1 | `nodeFromPath`, route context, `NodeBalancer` with delegate, static map + template + pool matching, passive health, first-attempt-only direct routing, `OnRoute`. The `basePath` fix from §9. Unit + fakeapi tests. | Done |
| 2 | `WithNodeResolver` and its refresh goroutine, for callers with a dynamic mapping of their own. | Done |
| 3 | Safe POST failover on provably-unsent transport errors (§7.2), via `WithSafePOSTFailover`. | Done |
| 4 | Deferred: guest→node affinity via `/cluster/resources`; pinning for websocket endpoints once they exist. | Deferred |

Phase 1 is self-contained and issues no requests of its own — the endpoint map comes
entirely from configuration the caller has already supplied. It is the whole feature
for anyone whose endpoint list is static or name-derivable, which is most users.

### 11.1 Implementation steps

Ordered so that each step compiles, passes tests and leaves the client working. Steps
1–3 are prerequisite fixes to existing code; 4–7 are new, self-contained and touch
nothing that already runs; only 8 and 9 modify the request path. Do them in order —
the later steps depend on the earlier ones, and reversing 8 and 9 leaves a balancer
installed that never receives a hint.

**1. Stop the balancer options from clobbering `basePath`** (`options.go`).
Delete `c.basePath = defaultBasePath` from `WithRoundRobin`, `WithWeightedRoundRobin`
and `WithSRVWeightedRoundRobin`. `New` already applies the default in its
`cfg.lb != nil` branch when the field is empty, which is the correct single place.
*Verify:* `WithBasePath("/pve/api2/json")` followed by `WithRoundRobin(...)` keeps the
custom prefix (§8.1).

**2. Record the pool's base URLs** (`client.go`, `options.go`).
Add an unexported `lbURLs []string` to `ClientConfig`; populate it in `WithURL`,
`WithRoundRobin` and `WithWeightedRoundRobin`. Needed because `resty.RoundRobin` and
`WeightedRoundRobin` expose no getter for their base URLs (§2), and pool matching
(§6.3) is the default endpoint source. While in `ClientConfig`, fix
`Client.ToRESTConfig` to copy `basePath` (and the new `lbURLs`) — it drops `basePath`
today while `ClientConfig.ToRESTConfig` keeps it (§9).

**3. Let `WithCACert` carry more than one CA** (`client.go`, `options.go`).
Make it variadic (or add `WithCACerts(...string)`), store `CACerts []string`, and pass
them through to the already-variadic `rc.SetRootCertificates(cfg.CACerts...)`. Document
that supplying any CA *replaces* the system pool — resty's `handleCAs` starts from a
fresh `x509.NewCertPool()` (§9). Independent of the rest; can land first — and did,
as its own `feat/multi-ca-cert` branch rather than part of this one.

**4. Routing hint** (new `loadbalancer.go`).
`nodeFromPath(p string) (string, bool)` per §4.1, plus the `route` struct, an
unexported context key, `withRoute` and `routeFrom`. Pure functions, no dependencies.
*Verify:* the `nodeFromPath` table test from §10.

**5. Endpoint registry** (new `loadbalancer_disco.go`).
`NodeEndpoint`, `NodeResolver`, and a `registry` resolving §6.1–6.3 in precedence
order. Normalise every base URL the way resty's `extractBaseURL` does — clear `Path`
and `RawQuery`, trim the trailing `/` — so a caller who passes
`https://pve1:8006/api2/json` in a static map gets the same endpoint as one who
passes `https://pve1:8006`.

**6. Health** (new `loadbalancer_health.go`).
The breaker from §7.1, keyed by **base URL**: failure counter, `MaxFailures`, open
state, and a recovery `time.Ticker` with a `done` channel. Copy the shutdown shape of
resty's `WeightedRoundRobin` (`close(done)` under a `sync.Once`, ticker stopped under
the lock) — `Ticker.Stop` does not close its channel, so the goroutine needs its own
signal.

**7. The balancer** (`loadbalancer.go`).
`NodeBalancer` implementing `resty.LoadBalancer` over a delegate: `NextWithContext`
exactly as §5, `Feedback` recording into health **and forwarding to the delegate**,
and `Close`.
*`Close` is the easy thing to get wrong:* resty closes only the balancer registered on
the client, so `NodeBalancer.Close` must close **both** its own ticker and the
delegate. A `WeightedRoundRobin` or SRV delegate owns goroutines that would otherwise
outlive `Client.Close`.
*Verify:* the §10 decision matrix against a stub delegate that records its calls.

**8. Options and construction** (`options.go`, `client.go`).
Add `nodeAffinityConfig`, `WithNodeAffinity` and the sub-options from §8, and a
`RouteDecision`/`OnRoute` hook defaulting to nil (checked before use — this is the hot
path). In `New`, after the option loop and before `rc.SetLoadBalancer`: if node
affinity is configured, build the delegate — `cfg.lb` if one was set, otherwise a
`resty.NewRoundRobin(cfg.BaseURL)` synthesised from the single URL with its path
stripped — wrap it in a `NodeBalancer`, and assign that to `cfg.lb`. **Do not touch
`basePath`**; the existing `cfg.lb != nil` branch defaults it correctly. No
post-construction wiring is needed, since no endpoint source calls the API (§6).

**9. Attach the hint** (`client.go`, `send`).
One conditional, immediately before `req.Execute(method, c.path(path))`:

```go
if node, ok := nodeFromPath(path); ok {
	req.SetContext(withRoute(req.Context(), &route{node: node}))
}
```

`send` is the single funnel for `do`, `doValues` and `Upload`, so this covers every
verb including the hand-built multipart upload. `path` here is the resource path
*before* `c.path` prepends `basePath`, which is what §4.1 expects. Requests issued
outside `send` — only `Client.ticket` — carry no route and are delegated, which is
correct for a cluster-wide endpoint.

**10. Tests, then docs.**
The §10 unit and fakeapi suites; then the `WithNodeAffinity` doc comment (leading with
the TLS and `WithCACert` notes from §9), a `README.md` section, and a pointer from
`docs/design.md` §3 to this document.

---

## 12. Alternatives considered

**Rewrite the URL in a request middleware instead of a balancer.** resty's
`AddRequestMiddleware` runs before `parseRequestURL` and could set an absolute URL,
bypassing the balancer entirely. Rejected: it duplicates base-URL composition, skips
`Feedback` (the request's `baseURL` would be empty), and breaks composition with the
user's own balancer. The balancer *is* the seam resty offers for "which host".

**A separate `*Client` per node, keyed by name.** Conceptually simple, and it sidesteps
the context hint. Rejected: it multiplies connection pools, ticket sessions and
retry/circuit state by the node count, and it forces callers to choose a client — the
whole point is that `client.Nodes("pve2")` already says everything needed.

**Teach `Client.send` to pick the host and pass an absolute URL.** Same objection as
the middleware option, plus it would make `path`/`basePath` handling conditional in a
third place.

**Ask resty for a path-aware `LoadBalancer`.** The right long-term fix (`Next(ctx,
*Request)`), worth raising upstream, but the context hint works today with no fork and
degrades cleanly if the interface ever widens.

---

## 13. Open questions

1. **Decided: on automatically for any configured balancer, no opt-out.** Every
   caller of `WithRoundRobin`, `WithWeightedRoundRobin`, `WithSRVWeightedRoundRobin`
   or `WithLoadBalancer` gets node affinity wrapped in for free, using zero-config
   pool matching (§6.3) as its endpoint source — `WithNodeAffinity()` is no longer
   what turns the feature on; it configures it (an explicit endpoint source, health
   tuning, `WithOnRoute`) or, for a client with no balancer at all (`WithURL` alone),
   is what turns it on. This reverses the original proposal (opt-in, on the grounds
   that rerouting traffic based on an *inferred* node map shouldn't happen silently)
   — the TLS objection that motivated "off" is retired once `WithCACert` supports
   multiple CAs (§9; ships as `feat/multi-ca-cert`, not part of this branch — this
   decision assumes that branch lands too); what's left is that pool matching degrades to a
   no-op rather than breaking on a miss (§5, §13 item 2), which makes "on by
   default" safe rather than merely convenient. A pool that pool matching
   misinterprets (an external LB/VIP whose host happens to collide with a real node
   name) is the one scenario this trades away silently; it was judged unlikely
   enough, and cheap enough to diagnose via `WithOnRoute`, not to warrant a
   never-used-in-practice opt-out.
2. **Decided: pool matching stays the default source, including for `WithURL`'s
   single URL** (§6.3, §11.1 step 2). It is inferred from data the user already
   supplied, and a wrong inference degrades to the current behaviour rather than
   breaking — concretely:
   - The common `WithURL` case is an external LB/VIP address that names no real
     node (`https://pve-vip.example.com:8006`). Pool matching simply misses for
     every node — no node-scoped request ever pins — and every request goes
     through the VIP exactly as it did before affinity was added. Nothing breaks;
     affinity is a no-op until a static map (§6.1) or template (§6.2) is added.
   - If instead the URL names a real node directly (a single-node deployment, or
     a caller pointing `WithURL` at one member's own address), pool matching
     succeeds for that one node — and since that same URL is also what `New`
     synthesises the delegate from (§11.1 step 8), the fallback for every other,
     unmapped node is that same one-entry pool. Direct and fallback coincide;
     there is nothing to special-case.
   - With a multi-member pool (`WithRoundRobin`/`WithWeightedRoundRobin`), a miss
     or a circuit-broken pinned node falls back to the delegate, which picks
     *a* member of that same pool — not necessarily a healthy one. `Feedback` is
     forwarded to the delegate unconditionally (§7.1) so a `WeightedRoundRobin`
     delegate's own health state does steer it away from members it has itself
     seen fail; `RoundRobin`'s `Feedback` is a no-op in resty, so a plain
     round-robin delegate falls back blindly, same as it would with affinity off.
     This is existing resty behaviour, not something node affinity adds to or
     needs to compensate for.
3. **Decided: `GET /nodes/{node}` (the node index) is left exactly as `nodeFromPath`
   already treats it — matched, so it would pin if it were ever called** (§4.1). No
   client method in this module issues that bare call (every `/nodes/{node}/...`
   method here names something under it — `status`, `config`, `qemu`, ...), so the
   question of whether pinning it is worth a special case doesn't arise in practice:
   there is no extra code to write either way, and none was added.
4. **Decided: whenever a balancer ends up installed — automatically (item 1) or via
   `WithNodeAffinity` on a `WithURL`-only client — the API prefix always comes from
   `basePath`** (§8.1), sourced from `WithBasePath` or from `WithURL`'s path,
   last-one-wins, defaulting to `/api2/json`. The open part is only whether the
   balancer options' unconditional `basePath` assignment (§9) may be changed without a
   deprecation note — it is a bug fix, but it is observable.
5. **Guest-level affinity (§1 non-goals) — wanted?** It would need a `vmid → node`
   cache invalidated on migration. Worth doing only if callers are making many
   `/cluster/resources`-shaped lookups anyway.
6. **Decided: direct routing is first-attempt-only, as an unexported constant**
   (§7.2), not a `WithMaxDirectAttempts` option. Recorded here so the question is not
   reopened in review.

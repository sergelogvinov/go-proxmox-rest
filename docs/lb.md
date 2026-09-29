# Load Balancing & Node Affinity — Usage Guide

How to configure client-side load balancing and node-affinity routing.

## 1. Choosing a balancer

Pick one. All of them distribute requests across the cluster's members.

```go
// Round-robin over a fixed list of endpoints.
proxmox.WithRoundRobin(
	"https://pve1.example.com:8006",
	"https://pve2.example.com:8006",
	"https://pve3.example.com:8006",
)

// Weighted round-robin, with per-host weight and failure threshold.
proxmox.WithWeightedRoundRobin(
	&resty.Host{BaseURL: "https://pve1.example.com:8006", Weight: 2},
	&resty.Host{BaseURL: "https://pve2.example.com:8006", Weight: 1},
)

// SRV-record discovery.
proxmox.WithSRVWeightedRoundRobin("pve", "tcp", "example.com", "https")

// Any custom resty.LoadBalancer implementation.
proxmox.WithLoadBalancer(myBalancer)
```

A client with none of these (just `proxmox.WithURL(...)`) talks to a single
endpoint and has no balancer installed at all.

## 2. Node affinity is automatic

Any of the four balancer options above gets node-affinity routing **for free,
with no extra call needed**: a request to `/nodes/{node}/...` is routed
directly at the node it names (by matching the node name against the
configured pool, see §3) instead of letting another cluster member's
`pveproxy` relay it. Everything else (`/cluster/*`, `/storage/*`, `/pools/*`,
`/version`, ...) keeps using the balancer as normal.

```go
client, err := proxmox.New(proxmox.ClientConfig{},
	proxmox.WithRoundRobin(
		"https://pve1.example.com:8006",
		"https://pve2.example.com:8006",
		"https://pve3.example.com:8006",
	),
	proxmox.WithTokenAuth(tokenID, secret),
	// Node affinity is already active here.
)
```

It never changes what a call returns, only which host serves it: if the
node's own endpoint is unknown or unreachable, the request is served by the
balancer exactly as it would be without affinity. Enabling it is safe to add
to (or already present in) any existing client.

`WithNodeAffinity(...)` is for the cases the automatic default doesn't cover:

- an explicit endpoint source instead of the inferred one (§3),
- tuning health/circuit-breaker behaviour (§4),
- a callback for observing routing decisions (§5),
- opting into POST failover (§6),
- enabling the feature on a client with **no** balancer — `WithURL` alone —
  where there's no pool to auto-wrap.

```go
client, err := proxmox.New(proxmox.ClientConfig{},
	proxmox.WithURL("https://pve-vip.example.com:8006/api2/json"),
	proxmox.WithNodeAffinity(
		proxmox.WithNodeEndpoints(map[string]string{
			"pve1": "https://10.0.0.1:8006",
			"pve2": "https://10.0.0.2:8006",
		}),
	),
)
```

Apply `WithNodeAffinity` after the balancer option.

## 3. Endpoint sources

How the client learns "node `pve2` is reachable at `https://10.0.0.2:8006`".
Consulted in this order; the first source that has an answer for a given
node wins:

1. **`WithNodeEndpoints(map[string]string)`** — an explicit node → base URL
   map. No I/O, no surprises. Recommended for production.
2. **`WithNodeEndpointTemplate(tmpl string)`** — a URL template with a
   `{node}` placeholder, e.g. `"https://{node}.pve.example.com:8006"`, for
   clusters where node names resolve in DNS. It matches *every* node name
   handed to it (it can't tell a real node from a typo), so once set it
   shadows the resolver and pool matching below for every node.
3. **`WithNodeResolver(r NodeResolver, refresh time.Duration)`** — a dynamic
   source (e.g. backed by a Kubernetes informer), refreshed on its own
   goroutine every `refresh` interval (default 60s if `refresh <= 0`). A
   failed refresh logs and keeps the previous map rather than clearing it.

   ```go
   type NodeResolver interface {
       Nodes(ctx context.Context) ([]proxmox.NodeEndpoint, error)
   }
   ```

   The resolver must be safe for concurrent use and must not route its own
   traffic through the client it is configuring.
4. **Pool matching** (default, zero-config) — derived from the base URLs
   already passed to `WithRoundRobin`/`WithWeightedRoundRobin`, matching
   case-insensitively against the full host and its first DNS label, e.g.
   `https://pve1.example.com:8006` matches node `pve1`. A bare IP address
   never matches. This is what makes the automatic case in §2 work with no
   configuration.

   `WithLoadBalancer` and `WithSRVWeightedRoundRobin` don't contribute a pool
   for this to match against, so with those, a request only pins once a
   static map, template, or resolver is configured via `WithNodeAffinity`.

## 4. Health

An endpoint that fails is temporarily taken out of direct routing —
purely from the outcome of real requests, never a background probe:

```go
proxmox.WithNodeAffinity(
	proxmox.WithNodeMaxFailures(5),        // default 5
	proxmox.WithNodeRecovery(120*time.Second), // default 120s
)
```

After `WithNodeMaxFailures` consecutive failures, that node's endpoint stops
being used for direct routing (delegated to the balancer instead) until
`WithNodeRecovery` has elapsed, at which point it's tried again.

## 5. Observing routing decisions

```go
proxmox.WithNodeAffinity(
	proxmox.WithOnRoute(func(node, baseURL string, d proxmox.RouteDecision) {
		metrics.Inc("proxmox_route", "decision", int(d))
	}),
)
```

`d` is one of `proxmox.RouteDirect`, `RouteUnknown`, `RouteUnhealthy`,
`RouteNotNodeScoped`. The default is a no-op.

## 6. POST behaviour on a dead node

By default, a `POST` against a node that just died fails with a plain
transport error — it is not retried. Every other call for that node then
falls back to the balancer once the failure count trips the breaker (§4).

To retry a `POST` automatically when it's provable the request never reached
the server (connection refused, DNS failure, TLS handshake failure — never
once bytes are on the wire):

```go
proxmox.WithNodeAffinity(
	proxmox.WithSafePOSTFailover(),
)
```

This needs at least one retry configured to have any effect. **Retries are off
by default** (`WithRetryCount` must be called explicitly with `n > 0` — an
unconfigured client makes exactly one attempt per request):

```go
proxmox.WithRetryCount(3),
```

## 7. TLS

Point the client at the cluster's CA and direct node connections verify like
any other:

```go
proxmox.WithCACert("/etc/pve/pve-root-ca.pem")
```

`WithCACert` accepts one or more paths. Supplying any CA **replaces** the
system trust store rather than adding to it, so a cluster fronted by a
VIP/ingress certificate that differs from the nodes' own PVE-CA-signed
certificates needs both listed:

```go
proxmox.WithCACert("/etc/ssl/certs/vip-ca.pem", "/etc/pve/pve-root-ca.pem")
```

## 8. Full example

```go
client, err := proxmox.New(proxmox.ClientConfig{},
	proxmox.WithRoundRobin(
		"https://pve1.example.com:8006",
		"https://pve2.example.com:8006",
		"https://pve3.example.com:8006",
	),
	proxmox.WithTokenAuth("root@pam!ci", secret),
	proxmox.WithCACert("/etc/pve/pve-root-ca.pem"),
	proxmox.WithRetryCount(3),
	proxmox.WithNodeAffinity(
		proxmox.WithNodeMaxFailures(3),
		proxmox.WithSafePOSTFailover(),
		proxmox.WithOnRoute(logRouteDecision),
	),
)
```

A runnable, self-contained demo is at
[examples/fakeapi-loadbalancer](../examples/fakeapi-loadbalancer) —
`go run ./examples/fakeapi-loadbalancer`.

# Proxmox REST API Client for Go

> **Status:** This project is under active development. Its API may change and
> breaking changes are possible. Do not use it in production yet.

`go-proxmox-rest` is a type-safe Go client for the [Proxmox VE REST API](https://pve.proxmox.com/pve-docs/api-viewer/).
It follows patterns familiar to users of the Kubernetes Go client: 
resource clients provide clear methods such as `Get`, `List`, `Create`, `Update`, and `Delete`.

## Features

- API-token and username/password authentication
- TLS, custom CA certificate, proxy, timeout, retry, and client-side load-balancing options
- Fluent access to cluster, node, pool, and storage resources
- Support for QEMU virtual machines, LXC containers, networking, storage,
  backups, firewall rules, HA, Ceph, replication, and tasks
- An in-memory fake Proxmox API for tests, without a running Proxmox cluster

## Quick start

Create a client with an API token. The URL must include the Proxmox API path:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
)

func main() {
	client, err := proxmox.New(
		proxmox.ClientConfig{},
		proxmox.WithURL(os.Getenv("PROXMOX_URL")),
		proxmox.WithTokenAuth(
			os.Getenv("PROXMOX_TOKEN_ID"),
			os.Getenv("PROXMOX_TOKEN_SECRET"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	version, err := client.Version(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Proxmox VE %s\\n", version.Version)
}
```

For username and password authentication, use
`proxmox.WithPasswordAuth("user@realm", "password")` instead of
`WithTokenAuth`. Password authentication creates and renews a Proxmox ticket
when needed.

For development systems with a self-signed certificate, you can add
`proxmox.WithInsecure(true)`. Do not use this option in production. Prefer
`proxmox.WithCACert("/path/to/ca.pem")` when you use a private certificate
authority.

## Using resources

The root client gives access to the main Proxmox API areas. For example:

```go
ctx := context.Background()

pools, err := client.Pools().List(ctx)
if err != nil {
	log.Fatal(err)
}

vm, err := client.Nodes("pve1").Qemu().Status(ctx, 100)
if err != nil {
	log.Fatal(err)
}

container, err := client.Nodes("pve1").LXC().Status(ctx, 101)
if err != nil {
	log.Fatal(err)
}

tasks, err := client.Nodes("pve1").Tasks().List(ctx, nil)
if err != nil {
	log.Fatal(err)
}
```

Available API areas include:

- `client.Cluster()` for cluster resources, jobs, backups, HA, firewall,
  Ceph, replication, mappings, and custom QEMU CPU models
- `client.Nodes("node")` for node status and configuration, QEMU VMs, LXC
  containers, tasks, storage, networking, hardware, Ceph, and backups
- `client.Pools()` for resource pools
- `client.Storage()` for cluster storage configuration
- `client.Version(ctx)` for the Proxmox VE version

Many write operations return a task identifier (UPID). Use the node task client
to inspect or wait for the task when the API operation is asynchronous.

## Load balancing and node affinity

`WithRoundRobin`, `WithWeightedRoundRobin`, `WithSRVWeightedRoundRobin`, and
`WithLoadBalancer` distribute requests across a Proxmox cluster's members.
Any of them also gets **node affinity automatically**: `/nodes/{node}/...`
requests are routed directly at the node they name (matched against the
configured pool by name), instead of letting another node's `pveproxy` relay
them — cheaper, and it turns a dead node into a plain transport error
instead of a relayed 5xx. It always falls back to the configured algorithm
when a node's endpoint is unknown or unreachable, so it never changes what a
call returns, only which host serves it.

```go
client, err := proxmox.New(proxmox.ClientConfig{},
	proxmox.WithRoundRobin(
		"https://pve1.example.com:8006",
		"https://pve2.example.com:8006",
		"https://pve3.example.com:8006",
	),
	proxmox.WithTokenAuth(tokenID, secret),
	// Node affinity is already active here — node endpoints are inferred
	// from the pool above. WithNodeAffinity is only needed to configure an
	// explicit endpoint source, health tuning, a routing-decision callback,
	// or to enable the feature on a client with no balancer (WithURL alone).
)
```

See [docs/node-lb.md](docs/node-lb.md) for the full design, including the
static-map and DNS-template endpoint sources, health/circuit-breaker
behavior, and the TLS caveats of connecting to nodes directly.

## Testing with the fake API

The `fakeapi` package starts an in-memory HTTP server that behaves like a small
Proxmox cluster. It is useful for testing code that uses this module without
access to a real cluster.

```go
func TestMyCode(t *testing.T) {
	cluster := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	client := cluster.Client(t)

	// Use client as you would use a client connected to Proxmox VE.
	_ = client
}
```

Read [the fake API design](docs/fakeapi.md) for its current behavior and
limitations. Runnable examples are in the [examples](examples) directory.

## Projects using this module

- [Proxmox CCM](https://github.com/sergelogvinov/proxmox-cloud-controller-manager)
- [Proxmox CSI](https://github.com/sergelogvinov/proxmox-csi-plugin)
- [Proxmox MCP server](https://github.com/sergelogvinov/proxmox-mcp)
- [Karpenter for Proxmox](https://github.com/sergelogvinov/karpenter-provider-proxmox)

## License

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

---

`Proxmox®` is a registered trademark of [Proxmox Server Solutions GmbH](https://www.proxmox.com/en/about/company).

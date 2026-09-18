// Package main demonstrates the fakeapi package (see docs/fakeapi.md): an
// in-memory, HTTP-level stand-in for a Proxmox VE cluster, driven through
// an ordinary *proxmox.Client exactly as if it were pointed at a real
// cluster.
//
// It builds a 3-node cluster (8 vCPU / 16 GiB per node), seeds each node
// with 2 QEMU VMs (2 vCPU / 4 GiB) and 1 LXC container (1 vCPU / 2 GiB),
// starts every guest, and prints the resulting cluster inventory —
// exactly the workflow a caller would use to smoke-test their own
// orchestration code against fakeapi instead of a live cluster.
//
// Run it with: go run ./examples/fakeapi
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/fakeapi"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
)

// guestRef identifies one seeded guest so the start/verify loops below
// can drive both QEMU VMs and LXC containers uniformly.
type guestRef struct {
	node string
	vmid int
	name string
	kind string // "qemu" or "lxc"
}

func main() {
	nodeNames := []string{"pve1", "pve2", "pve3"}

	// fakeapi.NewCluster/Cluster.Client take a testing.TB so the same
	// constructors work from `go test` (auto-closing via t.Cleanup) and,
	// as here, from a plain `go run`-able program: a nil interface value
	// is always legal to pass, even though testing.TB itself cannot be
	// implemented outside the testing package.
	cl := fakeapi.NewCluster(nil, fakeapi.WithNodes(nodeNames...))
	defer cl.Close()

	seedNodes(cl, nodeNames)
	guests := seedGuests(cl, nodeNames)

	c := cl.Client(nil)
	defer c.Close()

	ctx := context.Background()

	startGuests(ctx, c, guests)
	printInventory(ctx, c)
}

// seedNodes gives each node 8 vCPU / 16 GiB of reported capacity (fakeapi
// defaults to a much smaller 4 vCPU / 8 GiB node) and a local-lvm storage
// for the guests' root disks.
func seedNodes(cl *fakeapi.Cluster, nodeNames []string) {
	const (
		nodeCores  = 8
		nodeMemory = 16 << 30 // 16 GiB

		storageTotal = 1 << 40  // 1 TiB
		storageUsed  = 50 << 30 // 50 GiB already in use
		storageAvail = storageTotal - storageUsed
	)

	for _, name := range nodeNames {
		node := cl.Node(name)
		node.SetResources(nodeCores, nodeMemory)
		node.AddStorage("local-lvm", "lvm",
			fakeapi.WithCapacity(storageTotal, storageUsed, storageAvail))
	}
}

// seedGuests seeds 2 QEMU VMs (2 vCPU / 4 GiB) and 1 LXC container
// (1 vCPU / 2 GiB) per node, returning every guest it created so the
// caller can start and verify them.
func seedGuests(cl *fakeapi.Cluster, nodeNames []string) []guestRef {
	const (
		vmCores    = 2
		vmMemoryMB = 4096

		ctCores    = 1
		ctMemoryMB = 2048
	)

	var guests []guestRef
	vmid := 100

	for _, name := range nodeNames {
		node := cl.Node(name)

		for i := 1; i <= 2; i++ {
			id := vmid
			vmid++

			vmName := fmt.Sprintf("%s-web-%02d", name, i)
			cores := vmCores

			node.AddVM(id, &qemu.Config{
				Name:  vmName,
				Cores: &cores,
				Memory: &qemu.Memory{
					Current: new(vmMemoryMB),
				},
				SCSI: map[int]qemu.Drive{
					0: {File: fmt.Sprintf("local-lvm:vm-%d-disk-0", id), Size: "20G"},
				},
			})

			guests = append(guests, guestRef{node: name, vmid: id, name: vmName, kind: "qemu"})
		}

		id := vmid
		vmid++

		ctName := fmt.Sprintf("%s-ct-01", name)
		cores := ctCores
		memory := ctMemoryMB

		node.AddContainer(id, &lxc.Config{
			Hostname: ctName,
			Cores:    &cores,
			Memory:   &memory,
			RootFS: &lxc.RootFS{
				Volume: fmt.Sprintf("local-lvm:vm-%d-disk-0", id),
				Size:   "8G",
			},
		})

		guests = append(guests, guestRef{node: name, vmid: id, name: ctName, kind: "lxc"})
	}

	return guests
}

// startGuests starts every guest and confirms both its start task and
// its resulting runtime status. fakeapi completes tasks instantly by
// default (see fakeapi.WithManualTasks for the opposite), so
// waitForTask returns on its first check here — it's written as a poll
// loop anyway since that's the shape real code (and code exercising
// WithManualTasks) needs.
func startGuests(ctx context.Context, c *proxmox.Client, guests []guestRef) {
	for _, g := range guests {
		var (
			upid string
			err  error
		)

		switch g.kind {
		case "qemu":
			upid, err = c.Nodes(g.node).Qemu().Start(ctx, g.vmid, nil)
		case "lxc":
			upid, err = c.Nodes(g.node).LXC().Start(ctx, g.vmid, nil)
		}
		if err != nil {
			log.Fatalf("starting %s %s (vmid %d) on %s: %v", g.kind, g.name, g.vmid, g.node, err)
		}

		if err := waitForTask(ctx, c, g.node, upid); err != nil {
			log.Fatalf("waiting for %s %s (vmid %d) to start: %v", g.kind, g.name, g.vmid, err)
		}

		running, err := isRunning(ctx, c, g)
		if err != nil {
			log.Fatalf("checking status of %s %s (vmid %d): %v", g.kind, g.name, g.vmid, err)
		}
		if !running {
			log.Fatalf("%s %s (vmid %d) did not reach running state", g.kind, g.name, g.vmid)
		}

		fmt.Printf("started %-5s %-14s vmid=%-4d node=%s\n", g.kind, g.name, g.vmid, g.node)
	}
}

// waitForTask polls a task's status until it stops running, or ctx's
// deadline (none, here) / a bounded number of attempts is exhausted.
func waitForTask(ctx context.Context, c *proxmox.Client, node, upid string) error {
	for range 20 {
		status, err := c.Nodes(node).Tasks().Status(ctx, upid)
		if err != nil {
			return err
		}

		if status.Status != "running" {
			if status.ExitStatus != "OK" {
				return fmt.Errorf("task %s finished with %q", upid, status.ExitStatus)
			}
			return nil
		}

		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("task %s did not finish in time", upid)
}

func isRunning(ctx context.Context, c *proxmox.Client, g guestRef) (bool, error) {
	switch g.kind {
	case "qemu":
		status, err := c.Nodes(g.node).Qemu().Status(ctx, g.vmid)
		if err != nil {
			return false, err
		}
		return status.Status == qemu.VMStatusRunning, nil

	case "lxc":
		status, err := c.Nodes(g.node).LXC().Status(ctx, g.vmid)
		if err != nil {
			return false, err
		}
		return status.Status == lxc.StateRunning, nil
	}

	return false, fmt.Errorf("unknown guest kind %q", g.kind)
}

// printInventory prints the cluster-wide view a caller would use to
// confirm "everything is up": GET /cluster/resources, filtered to nodes,
// then storages, then guests.
func printInventory(ctx context.Context, c *proxmox.Client) {
	fmt.Println()
	fmt.Println("cluster inventory (GET /cluster/resources):")

	nodes, err := c.Cluster().Resources().List(ctx, cluster.ListFilter{Type: cluster.ResourceTypeNode})
	if err != nil {
		log.Fatalf("listing nodes: %v", err)
	}
	for _, n := range nodes {
		fmt.Printf("  node  %-6s status=%-8s cpus=%-2d mem=%dGiB\n",
			n.Name, n.Status, n.MaxCPU, n.MaxMem>>30)
	}

	storages, err := c.Cluster().Resources().List(ctx, cluster.ListFilter{Type: cluster.ResourceTypeStorage})
	if err != nil {
		log.Fatalf("listing storages: %v", err)
	}
	for _, s := range storages {
		fmt.Printf("  store %-6s status=%-8s node=%-5s type=%-4s used=%dGiB/%dGiB\n",
			s.Storage, s.Status, s.Node, s.PluginType, s.Disk>>30, s.MaxDisk>>30)
	}

	guests, err := c.Cluster().Resources().List(ctx, cluster.ListFilter{Type: cluster.ResourceTypeVM})
	if err != nil {
		log.Fatalf("listing guests: %v", err)
	}
	for _, g := range guests {
		fmt.Printf("  %-4s  %-14s vmid=%-4d node=%-5s status=%-8s cpus=%d mem=%dMiB\n",
			g.Type, g.Name, g.VMID, g.Node, g.Status, g.MaxCPU, g.MaxMem>>20)
	}

	fmt.Println()
	fmt.Printf("%d nodes, %d storages, %d guests, all running.\n", len(nodes), len(storages), len(guests))
}

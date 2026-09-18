/*
Copyright 2026 Proxmox Community.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fakeapi_test

import (
	"context"
	"testing"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/fakeapi"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
)

func TestClusterStatusAndResources(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1", "pve2", "pve3"))
	c := cl.Client(t)
	ctx := context.Background()

	status, err := c.Cluster().Status(ctx)
	if err != nil {
		t.Fatalf("cluster status: %v", err)
	}
	if status.Quorate != 1 {
		t.Fatalf("expected quorate cluster, got %+v", status)
	}
	if len(status.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d: %+v", len(status.Nodes), status.Nodes)
	}

	cl.Node("pve1").AddVM(100, &qemu.Config{Name: "web-01"})

	resources, err := c.Cluster().Resources().List(ctx, cluster.ListFilter{Type: cluster.ResourceTypeVM})
	if err != nil {
		t.Fatalf("cluster resources: %v", err)
	}
	if len(resources) != 1 || resources[0].VMID != 100 || resources[0].Name != "web-01" {
		t.Fatalf("unexpected resources: %+v", resources)
	}
}

func TestQemuLifecycle(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	cores := 2
	cl.Node("pve1").AddVM(100, &qemu.Config{
		Name:  "web-01",
		Cores: &cores,
		SCSI:  map[int]qemu.Drive{0: {File: "local-lvm:vm-100-disk-0", Size: "20G"}},
	})

	c := cl.Client(t)
	ctx := context.Background()

	cfg, err := c.Nodes("pve1").Qemu().Config(ctx, 100, nil)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if cfg.Name != "web-01" || cfg.SCSI[0].File != "local-lvm:vm-100-disk-0" {
		t.Fatalf("unexpected config: %+v", cfg)
	}

	status, err := c.Nodes("pve1").Qemu().Status(ctx, 100)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != qemu.VMStatusStopped {
		t.Fatalf("expected stopped, got %s", status.Status)
	}

	upid, err := c.Nodes("pve1").Qemu().Start(ctx, 100, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if upid == "" {
		t.Fatal("expected non-empty UPID")
	}

	taskStatus, err := c.Nodes("pve1").Tasks().Status(ctx, upid)
	if err != nil {
		t.Fatalf("task status: %v", err)
	}
	if taskStatus.ExitStatus != "OK" {
		t.Fatalf("expected OK task, got %+v", taskStatus)
	}

	status, err = c.Nodes("pve1").Qemu().Status(ctx, 100)
	if err != nil {
		t.Fatalf("status after start: %v", err)
	}
	if status.Status != qemu.VMStatusRunning {
		t.Fatalf("expected running, got %s", status.Status)
	}
}

func TestLXCLifecycle(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	cores := 1
	cl.Node("pve1").AddContainer(200, &lxc.Config{
		Hostname: "ct-01",
		Cores:    &cores,
		RootFS:   &lxc.RootFS{Volume: "local-lvm:vm-200-disk-0", Size: "8G"},
	})

	c := cl.Client(t)
	ctx := context.Background()

	if _, err := c.Nodes("pve1").LXC().Config(ctx, 999, nil); !proxmox.IsNotFound(err) {
		t.Fatalf("expected IsNotFound for an unseeded container, got %v", err)
	}

	upid, err := c.Nodes("pve1").LXC().Start(ctx, 200, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	status, err := c.Nodes("pve1").LXC().Status(ctx, 200)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != lxc.StateRunning {
		t.Fatalf("expected running, got %s", status.Status)
	}
	if status.Name != "ct-01" {
		t.Fatalf("expected hostname to round-trip, got %q", status.Name)
	}
	_ = upid
}

func TestStorageContentLifecycle(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	cl.Node("pve1").AddStorage("local-lvm", "lvm", fakeapi.WithCapacity(500<<30, 100<<30, 400<<30))

	c := cl.Client(t)
	ctx := context.Background()

	volid, err := c.Nodes("pve1").Storage().Content().Create(ctx, "local-lvm", &storage.CreateVolumeOptions{
		Filename: "vm-100-disk-1", VMID: 100, Size: "10G", Format: "raw",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	vols, err := c.Nodes("pve1").Storage().Content().List(ctx, "local-lvm", &storage.ContentListOptions{VMID: 100})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vols) != 1 || vols[0].VolID != volid {
		t.Fatalf("unexpected volumes: %+v", vols)
	}

	if _, err := c.Nodes("pve1").Storage().Content().Delete(ctx, "local-lvm", volid, 0); err != nil {
		t.Fatalf("delete: %v", err)
	}

	vols, err = c.Nodes("pve1").Storage().Content().List(ctx, "local-lvm", nil)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(vols) != 0 {
		t.Fatalf("expected no volumes after delete, got %+v", vols)
	}
}

func TestManualTasks(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"), fakeapi.WithManualTasks())
	cl.Node("pve1").AddVM(100, &qemu.Config{Name: "web-01"})

	c := cl.Client(t)
	ctx := context.Background()

	upid, err := c.Nodes("pve1").Qemu().Start(ctx, 100, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	status, err := c.Nodes("pve1").Qemu().Status(ctx, 100)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != qemu.VMStatusStopped {
		t.Fatalf("expected the manual task to not have applied yet, got %s", status.Status)
	}

	cl.Tasks().Complete(upid)

	status, err = c.Nodes("pve1").Qemu().Status(ctx, 100)
	if err != nil {
		t.Fatalf("status after complete: %v", err)
	}
	if status.Status != qemu.VMStatusRunning {
		t.Fatalf("expected running after Complete, got %s", status.Status)
	}
}

func TestFailNode(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1", "pve2", "pve3"))
	c := cl.Client(t)
	ctx := context.Background()

	cl.FailNode("pve3", fakeapi.FailureUnreachable)

	if _, err := c.Nodes("pve3").Status(ctx); !proxmox.IsUnexpected(err) {
		t.Fatalf("expected IsUnexpected error, got %v", err)
	}

	status, err := c.Cluster().Status(ctx)
	if err != nil {
		t.Fatalf("cluster status: %v", err)
	}
	if status.Quorate != 1 {
		t.Fatalf("expected 2/3 nodes to still form quorum, got %+v", status)
	}

	pve3Online := -1
	for _, n := range status.Nodes {
		if n.Name == "pve3" {
			pve3Online = n.Online
		}
	}
	if pve3Online != 0 {
		t.Fatalf("expected pve3 to be reported offline, got online=%d", pve3Online)
	}

	cl.RecoverNode("pve3")
	if _, err := c.Nodes("pve3").Status(ctx); err != nil {
		t.Fatalf("expected pve3 to recover, got %v", err)
	}
}

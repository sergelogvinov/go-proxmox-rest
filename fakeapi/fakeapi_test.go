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
	"errors"
	"testing"
	"time"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/fakeapi"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/tasks"
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

func TestQemuCreate(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	c := cl.Client(t)
	ctx := context.Background()

	cores := 2
	memory := 2048
	upid, err := c.Nodes("pve1").Qemu().Create(ctx, &qemu.CreateOptions{
		VMID: 300,
		Config: qemu.Config{
			Name:   "created-01",
			Cores:  &cores,
			Memory: &qemu.Memory{Current: &memory},
			SCSI:   map[int]qemu.Drive{0: {File: "local-lvm:vm-300-disk-0", Size: "10G"}},
		},
		Start: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
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

	cfg, err := c.Nodes("pve1").Qemu().Config(ctx, 300, nil)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if cfg.Name != "created-01" || cfg.SCSI[0].File != "local-lvm:vm-300-disk-0" {
		t.Fatalf("unexpected config: %+v", cfg)
	}

	status, err := c.Nodes("pve1").Qemu().Status(ctx, 300)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != qemu.VMStatusRunning {
		t.Fatalf("expected running (Start: true), got %s", status.Status)
	}

	resources, err := c.Cluster().Resources().List(ctx, cluster.ListFilter{Type: cluster.ResourceTypeVM, VMID: 300})
	if err != nil {
		t.Fatalf("cluster resources: %v", err)
	}
	if len(resources) != 1 || resources[0].Name != "created-01" {
		t.Fatalf("expected the new guest in cluster resources, got %+v", resources)
	}

	if _, err := c.Nodes("pve1").Qemu().Create(ctx, &qemu.CreateOptions{VMID: 300}); err == nil {
		t.Fatal("expected an error creating a guest with an already-used vmid")
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

func TestLXCCreate(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	c := cl.Client(t)
	ctx := context.Background()

	cores := 2
	memory := 512
	upid, err := c.Nodes("pve1").LXC().Create(ctx, &lxc.CreateOptions{
		VMID:       300,
		OSTemplate: "local:vztmpl/debian-12-standard_12.2-1_amd64.tar.zst",
		Config: lxc.Config{
			Hostname: "created-01",
			Cores:    &cores,
			Memory:   &memory,
			RootFS:   &lxc.RootFS{Volume: "local-lvm:vm-300-disk-0", Size: "8G"},
		},
		Start: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
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

	cfg, err := c.Nodes("pve1").LXC().Config(ctx, 300, nil)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if cfg.Hostname != "created-01" || cfg.RootFS == nil || cfg.RootFS.Volume != "local-lvm:vm-300-disk-0" {
		t.Fatalf("unexpected config: %+v", cfg)
	}

	status, err := c.Nodes("pve1").LXC().Status(ctx, 300)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != lxc.StateRunning {
		t.Fatalf("expected running (Start: true), got %s", status.Status)
	}

	resources, err := c.Cluster().Resources().List(ctx, cluster.ListFilter{Type: cluster.ResourceTypeVM, VMID: 300})
	if err != nil {
		t.Fatalf("cluster resources: %v", err)
	}
	if len(resources) != 1 || resources[0].Name != "created-01" {
		t.Fatalf("expected the new container in cluster resources, got %+v", resources)
	}

	if _, err := c.Nodes("pve1").LXC().Create(ctx, &lxc.CreateOptions{
		VMID:       300,
		OSTemplate: "local:vztmpl/debian-12-standard_12.2-1_amd64.tar.zst",
	}); err == nil {
		t.Fatal("expected an error creating a container with an already-used vmid")
	}
}

func TestStorageContentLifecycle(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	cl.Node("pve1").AddStorage("local-lvm", "lvm", fakeapi.WithCapacity(500<<30, 100<<30, 400<<30))

	c := cl.Client(t)
	ctx := context.Background()

	volid, err := c.Nodes("pve1").Storage().Content("local-lvm").Create(ctx, &storage.CreateVolumeOptions{
		Filename: "vm-100-disk-1", VMID: 100, Size: "10G", Format: "raw",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	vols, err := c.Nodes("pve1").Storage().Content("local-lvm").List(ctx, &storage.ContentListOptions{VMID: 100})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vols) != 1 || vols[0].VolID != volid {
		t.Fatalf("unexpected volumes: %+v", vols)
	}

	if _, err := c.Nodes("pve1").Storage().Content("local-lvm").Delete(ctx, volid, 0); err != nil {
		t.Fatalf("delete: %v", err)
	}

	vols, err = c.Nodes("pve1").Storage().Content("local-lvm").List(ctx, nil)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(vols) != 0 {
		t.Fatalf("expected no volumes after delete, got %+v", vols)
	}
}

func TestStorageContentCopy(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1", "pve2"))
	cl.Node("pve1").AddStorage("shared", "nfs", fakeapi.WithCapacity(500<<30, 100<<30, 400<<30),
		fakeapi.WithVolume(storage.Volume{VolID: "shared:vm-100-disk-0", VMID: 100, Format: "raw", Size: 10 << 30}))
	cl.Node("pve2").AddStorage("shared", "nfs", fakeapi.WithCapacity(500<<30, 100<<30, 400<<30))

	c := cl.Client(t)
	ctx := context.Background()

	// Same-node copy to a new volume name.
	upid, err := c.Nodes("pve1").Storage().Content("shared").Copy(ctx, "vm-100-disk-0", &storage.CopyOptions{
		Target: "vm-100-disk-1",
	})
	if err != nil {
		t.Fatalf("copy: %v", err)
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

	vols, err := c.Nodes("pve1").Storage().Content("shared").List(ctx, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vols) != 2 {
		t.Fatalf("expected source and copy on pve1, got %+v", vols)
	}

	// Cross-node move to pve2's copy of the same shared storage.
	if _, err := c.Nodes("pve1").Storage().Content("shared").Copy(ctx, "vm-100-disk-0", &storage.CopyOptions{
		Target:     "vm-100-disk-2",
		TargetNode: "pve2",
	}); err != nil {
		t.Fatalf("cross-node copy: %v", err)
	}

	vols, err = c.Nodes("pve2").Storage().Content("shared").List(ctx, nil)
	if err != nil {
		t.Fatalf("list on pve2: %v", err)
	}
	if len(vols) != 1 || vols[0].VolID != "shared:vm-100-disk-2" {
		t.Fatalf("expected the copy on pve2, got %+v", vols)
	}

	// A source volume that doesn't exist.
	if _, err := c.Nodes("pve1").Storage().Content("shared").Copy(ctx, "does-not-exist", &storage.CopyOptions{
		Target: "vm-100-disk-3",
	}); !proxmox.IsNotFound(err) {
		t.Fatalf("expected IsNotFound copying an absent volume, got %v", err)
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

func TestTasksWait(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	cl.Node("pve1").AddVM(100, &qemu.Config{Name: "web-01"})

	c := cl.Client(t)
	ctx := context.Background()

	// Instant mode: the task is already stopped by the time Start
	// returns, so Wait must return immediately without ever polling.
	upid, err := c.Nodes("pve1").Qemu().Start(ctx, 100, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := c.Nodes("pve1").Tasks().Wait(ctx, upid, nil); err != nil {
		t.Fatalf("wait: %v", err)
	}
}

func TestTasksWaitFailed(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"), fakeapi.WithManualTasks())
	cl.Node("pve1").AddVM(100, &qemu.Config{Name: "web-01"})

	c := cl.Client(t)
	ctx := context.Background()

	upid, err := c.Nodes("pve1").Qemu().Start(ctx, 100, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	waitErr := make(chan error, 1)
	go func() {
		waitErr <- c.Nodes("pve1").Tasks().Wait(ctx, upid, &tasks.WaitOptions{
			PollInterval: 20 * time.Millisecond,
			Timeout:      2 * time.Second,
		})
	}()

	time.Sleep(50 * time.Millisecond)
	cl.Tasks().Fail(upid, "boom")

	err = <-waitErr

	var failed *tasks.FailedError
	if !errors.As(err, &failed) {
		t.Fatalf("expected a *tasks.FailedError, got %v", err)
	}
	if failed.UPID != upid || failed.ExitStatus != "boom" {
		t.Fatalf("unexpected FailedError: %+v", failed)
	}
}

func TestTasksWaitTimeout(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"), fakeapi.WithManualTasks())
	cl.Node("pve1").AddVM(100, &qemu.Config{Name: "web-01"})

	c := cl.Client(t)
	ctx := context.Background()

	// Never completed: Wait must give up once its Timeout elapses.
	upid, err := c.Nodes("pve1").Qemu().Start(ctx, 100, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	err = c.Nodes("pve1").Tasks().Wait(ctx, upid, &tasks.WaitOptions{
		PollInterval: 10 * time.Millisecond,
		Timeout:      50 * time.Millisecond,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected an error wrapping context.DeadlineExceeded, got %v", err)
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

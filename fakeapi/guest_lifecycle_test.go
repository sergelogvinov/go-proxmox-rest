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
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/fakeapi"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu/firewall"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
	"github.com/sergelogvinov/go-proxmox-rest/pools"
	"github.com/sergelogvinov/go-proxmox-rest/task"
)

// seedTemplate seeds VM template 100 on pve1 with one disk and a native
// cloud-init drive on local-lvm, one NIC and an SMBIOS UUID.
func seedTemplate(cl *fakeapi.Cluster) {
	template := true
	cl.Node("pve1").AddStorage("local-lvm", "lvmthin",
		fakeapi.WithVolume(storage.Volume{VolID: "local-lvm:base-100-disk-0", VMID: 100}),
		fakeapi.WithVolume(storage.Volume{VolID: "local-lvm:vm-100-cloudinit", VMID: 100}))
	cl.Node("pve1").AddStorage("local", "dir")
	cl.Node("pve1").AddVM(100, &qemu.Config{
		Name:     "template",
		Template: &template,
		SCSI:     map[int]qemu.Drive{0: {File: "local-lvm:base-100-disk-0", Size: "10G"}},
		IDE:      map[int]qemu.Drive{0: {File: "local-lvm:vm-100-cloudinit", Media: "cdrom"}},
		Net:      map[int]qemu.Net{0: {Model: "virtio", MACAddr: "AA:BB:CC:DD:EE:FF", Bridge: "vmbr0"}},
		SMBios1:  &qemu.SMBios1{UUID: "11111111-1111-1111-1111-111111111111"},
	})
}

func waitTask(t *testing.T, tk *task.Task, err error) error {
	t.Helper()

	if err != nil {
		t.Fatalf("starting task: %v", err)
	}

	return tk.Wait(context.Background(), &task.WaitOptions{Timeout: 5 * time.Second})
}

func volumes(t *testing.T, c *proxmox.Client, storageID string) []string {
	t.Helper()

	vols, err := c.Nodes("pve1").Storage().Content(storageID).List(context.Background(), nil)
	if err != nil {
		t.Fatalf("listing %s: %v", storageID, err)
	}

	ids := make([]string, len(vols))
	for i, v := range vols {
		ids[i] = v.VolID
	}

	return ids
}

func TestQemuClone(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"), fakeapi.WithPool("karpenter"))
	seedTemplate(cl)

	c := cl.Client(t)
	ctx := context.Background()
	full := true

	tk, err := c.Nodes("pve1").Qemu().Clone(ctx, 100, &qemu.CloneOptions{
		NewID: 101, Name: "node-1", Full: &full, Pool: "karpenter", Storage: "local-lvm",
	})
	if err := waitTask(t, tk, err); err != nil {
		t.Fatalf("clone: %v", err)
	}

	cfg, err := c.Nodes("pve1").Qemu().Config(ctx, 101, nil)
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	if cfg.Name != "node-1" || cfg.Template != nil && *cfg.Template {
		t.Fatalf("unexpected clone identity: name=%q template=%v", cfg.Name, cfg.Template)
	}
	if cfg.SCSI[0].File != "local-lvm:vm-101-disk-0" || cfg.SCSI[0].Size != "10G" {
		t.Fatalf("clone disk not copied: %+v", cfg.SCSI[0])
	}
	if cfg.Net[0].MACAddr == "AA:BB:CC:DD:EE:FF" || cfg.Net[0].Bridge != "vmbr0" {
		t.Fatalf("clone NIC should get a new MAC on the same bridge: %+v", cfg.Net[0])
	}
	if cfg.SMBios1 == nil || cfg.SMBios1.UUID == "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("clone should get a new SMBIOS UUID: %+v", cfg.SMBios1)
	}

	if cfg.IDE[0].File != "local-lvm:vm-101-cloudinit" {
		t.Fatalf("clone should get its own cloud-init drive: %+v", cfg.IDE[0])
	}

	if vols := volumes(t, c, "local-lvm"); !slices.Contains(vols, "local-lvm:vm-101-disk-0") ||
		!slices.Contains(vols, "local-lvm:vm-101-cloudinit") {
		t.Fatalf("clone volumes missing: %v", vols)
	}

	pool, err := c.Pools().Get(ctx, "karpenter")
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	if len(pool.Members) != 1 || pool.Members[0].VMID != 101 {
		t.Fatalf("clone should join its pool: %+v", pool.Members)
	}

	resources, err := c.Cluster().Resources().List(ctx, cluster.ListFilter{Type: cluster.ResourceTypeVM})
	if err != nil {
		t.Fatalf("resources: %v", err)
	}
	for _, r := range resources {
		switch r.VMID {
		case 100:
			if r.Template != 1 {
				t.Fatalf("template should be flagged: %+v", r)
			}
		case 101:
			if r.Pool != "karpenter" || r.Template != 0 {
				t.Fatalf("clone resource: %+v", r)
			}
		}
	}

	_, err = c.Nodes("pve1").Qemu().Clone(ctx, 100, &qemu.CloneOptions{NewID: 101, Full: &full})
	if !proxmox.IsUnexpected(err) {
		t.Fatalf("cloning onto an existing vmid should fail, got %v", err)
	}
}

func TestQemuDelete(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"), fakeapi.WithPool("karpenter"))
	seedTemplate(cl)
	cl.Node("pve1").AddStorage("iso", "dir", fakeapi.WithVolume(storage.Volume{VolID: "iso:iso/seed.iso"}))

	c := cl.Client(t)
	ctx := context.Background()
	full := true
	vm := c.Nodes("pve1").Qemu()

	tk, err := vm.Clone(ctx, 100, &qemu.CloneOptions{NewID: 101, Full: &full, Pool: "karpenter"})
	if err := waitTask(t, tk, err); err != nil {
		t.Fatalf("clone: %v", err)
	}

	tk, err = vm.AttachISO(ctx, 101, &qemu.AttachISOOptions{Drive: "ide2", Volume: "iso:iso/seed.iso"})
	if err := waitTask(t, tk, err); err != nil {
		t.Fatalf("attach: %v", err)
	}

	tk, err = vm.Start(ctx, 101, nil)
	if err := waitTask(t, tk, err); err != nil {
		t.Fatalf("start: %v", err)
	}

	if _, err := vm.Delete(ctx, 101, nil); !proxmox.IsUnexpected(err) {
		t.Fatalf("deleting a running VM should fail, got %v", err)
	}

	tk, err = vm.Stop(ctx, 101, nil)
	if err := waitTask(t, tk, err); err != nil {
		t.Fatalf("stop: %v", err)
	}

	tk, err = vm.Delete(ctx, 101, nil)
	if err := waitTask(t, tk, err); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := vm.Config(ctx, 101, nil); !proxmox.IsNotFound(err) {
		t.Fatalf("deleted VM should be gone, got %v", err)
	}
	if vols := volumes(t, c, "local-lvm"); slices.Contains(vols, "local-lvm:vm-101-disk-0") ||
		slices.Contains(vols, "local-lvm:vm-101-cloudinit") {
		t.Fatalf("deleted VM's own disks should be removed: %v", vols)
	}
	if vols := volumes(t, c, "iso"); !slices.Contains(vols, "iso:iso/seed.iso") {
		t.Fatalf("an attached ISO is not the VM's own and should stay: %v", vols)
	}

	pool, err := c.Pools().Get(ctx, "karpenter")
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	if len(pool.Members) != 0 {
		t.Fatalf("deleted VM should leave its pool: %+v", pool.Members)
	}
}

func TestClusterNextID(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	cl.Node("pve1").AddVM(100, &qemu.Config{})

	c := cl.Client(t)
	ctx := context.Background()

	if id, err := c.Cluster().NextID(ctx, 0); err != nil || id != 101 {
		t.Fatalf("next free id: got %d, %v", id, err)
	}
	if id, err := c.Cluster().NextID(ctx, 500); err != nil || id != 500 {
		t.Fatalf("free hint: got %d, %v", id, err)
	}

	_, err := c.Cluster().NextID(ctx, 100)

	var apiErr *proxmox.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("used hint should be a 400, got %v", err)
	}
}

func TestFirewallRules(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	cl.Node("pve1").AddVM(100, &qemu.Config{})

	c := cl.Client(t)
	ctx := context.Background()
	fw := c.Nodes("pve1").Qemu().Firewall()
	rules := fw.Rules(100)

	actions := func() []string {
		t.Helper()

		list, err := rules.List(ctx)
		if err != nil {
			t.Fatalf("list: %v", err)
		}

		out := make([]string, len(list))
		for i, r := range list {
			if r.Pos != i {
				t.Fatalf("rule %d reports pos %d", i, r.Pos)
			}
			out[i] = r.Action
		}

		return out
	}

	for _, a := range []string{"a", "b", "c"} {
		if err := rules.Create(ctx, &firewall.RuleOptions{Type: firewall.RuleTypeGroup, Action: a}); err != nil {
			t.Fatalf("create %s: %v", a, err)
		}
	}

	if got := actions(); !slices.Equal(got, []string{"c", "b", "a"}) {
		t.Fatalf("create should insert at the top: %v", got)
	}

	rule, err := rules.Get(ctx, 0)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rule.Enable != 0 {
		t.Fatalf("a rule created without enable should be disabled: %+v", rule)
	}

	moveto := 3
	if err := rules.Update(ctx, 0, &firewall.RuleOptions{MoveTo: &moveto}); err != nil {
		t.Fatalf("move: %v", err)
	}
	if got := actions(); !slices.Equal(got, []string{"b", "a", "c"}) {
		t.Fatalf("moveto past the end should append: %v", got)
	}

	if err := rules.Delete(ctx, 0, ""); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got := actions(); !slices.Equal(got, []string{"a", "c"}) {
		t.Fatalf("delete should renumber: %v", got)
	}

	if err := rules.Delete(ctx, 2, ""); !proxmox.IsUnexpected(err) {
		t.Fatalf("deleting past the end should fail, got %v", err)
	}

	if err := fw.Options(100).Update(ctx, &firewall.Options{Enable: 1, PolicyIn: firewall.PolicyDrop}); err != nil {
		t.Fatalf("options update: %v", err)
	}

	opts, err := fw.Options(100).Get(ctx)
	if err != nil {
		t.Fatalf("options get: %v", err)
	}
	if opts.Enable != 1 || opts.PolicyIn != firewall.PolicyDrop {
		t.Fatalf("options not stored: %+v", opts)
	}
}

func TestPools(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"), fakeapi.WithPool("a"), fakeapi.WithPool("b"))
	cl.Node("pve1").AddVM(100, &qemu.Config{Name: "web"})

	c := cl.Client(t)
	ctx := context.Background()
	remove := true
	allowMove := true

	members := func(id string) []int {
		t.Helper()

		p, err := c.Pools().Get(ctx, id)
		if err != nil {
			t.Fatalf("get %s: %v", id, err)
		}

		out := make([]int, 0, len(p.Members))
		for _, m := range p.Members {
			out = append(out, m.VMID)
		}

		return out
	}

	if err := c.Pools().Update(ctx, "a", &pools.UpdateOptions{VMIDs: []int{100}}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if got := members("a"); !slices.Equal(got, []int{100}) {
		t.Fatalf("pool a: %v", got)
	}

	if err := c.Pools().Update(ctx, "b", &pools.UpdateOptions{VMIDs: []int{100}}); !proxmox.IsUnexpected(err) {
		t.Fatalf("adding a member of another pool should fail, got %v", err)
	}
	if err := c.Pools().Update(ctx, "b", &pools.UpdateOptions{VMIDs: []int{100}, AllowMove: &allowMove}); err != nil {
		t.Fatalf("move: %v", err)
	}
	if a, b := members("a"), members("b"); len(a) != 0 || !slices.Equal(b, []int{100}) {
		t.Fatalf("allow-move should move the guest: a=%v b=%v", a, b)
	}

	if err := c.Pools().Update(ctx, "b", &pools.UpdateOptions{VMIDs: []int{100}, Remove: &remove}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if got := members("b"); len(got) != 0 {
		t.Fatalf("pool b after remove: %v", got)
	}

	if _, err := c.Pools().Get(ctx, "missing"); !proxmox.IsNotFound(err) {
		t.Fatalf("unknown pool should be not found, got %v", err)
	}
}

func TestStorageUpload(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	cl.Node("pve1").AddStorage("local", "dir")

	c := cl.Client(t)
	ctx := context.Background()

	upload := func() (*task.Task, error) {
		return c.Nodes("pve1").Storage().Upload(ctx, "local", &storage.UploadOptions{
			Content: "iso", Filename: "seed.iso", File: strings.NewReader("data"),
		})
	}

	tk, err := upload()
	if err := waitTask(t, tk, err); err != nil {
		t.Fatalf("upload: %v", err)
	}

	vols, err := c.Nodes("pve1").Storage().Content("local").List(ctx, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vols) != 1 || vols[0].VolID != "local:iso/seed.iso" || vols[0].Size != 4 {
		t.Fatalf("uploaded volume: %+v", vols)
	}

	if _, err := upload(); err == nil {
		t.Fatalf("uploading over an existing file should fail")
	}
}

func TestFailRequests(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	seedTemplate(cl)

	c := cl.Client(t)
	ctx := context.Background()
	full := true
	vm := c.Nodes("pve1").Qemu()
	clone := func(newID int) (*task.Task, error) {
		return vm.Clone(ctx, 100, &qemu.CloneOptions{NewID: newID, Full: &full})
	}

	cl.FailRequests("POST /nodes/{node}/qemu/{vmid}/clone", fakeapi.WithFaultMessage("boom"))

	if _, err := clone(101); !proxmox.IsUnexpected(err) || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("faulted clone should fail with the fault message, got %v", err)
	}
	if _, err := vm.Config(ctx, 101, nil); !proxmox.IsNotFound(err) {
		t.Fatalf("a faulted request should change nothing, got %v", err)
	}

	if _, err := vm.Config(ctx, 100, nil); err != nil {
		t.Fatalf("other routes should be unaffected: %v", err)
	}

	cl.FailRequests("POST /nodes/{node}/qemu/{vmid}/clone", fakeapi.AsFailedTask())

	tk, err := clone(101)
	if err := waitTask(t, tk, err); err == nil {
		t.Fatalf("clone task should fail")
	}
	if _, err := vm.Config(ctx, 101, nil); err != nil {
		t.Fatalf("a failed task keeps its work: %v", err)
	}

	cl.FailRequests("PUT /nodes/{node}/qemu/{vmid}/config")

	if err := vm.UpdateConfig(ctx, 100, &qemu.Config{Name: "x"}); !proxmox.IsUnexpected(err) {
		t.Fatalf("a fault on a bare-registered route should match its method, got %v", err)
	}
	if _, err := vm.Config(ctx, 100, nil); err != nil {
		t.Fatalf("GET on the same path should be unaffected: %v", err)
	}

	cl.ClearFaults()

	tk, err = clone(102)
	if err := waitTask(t, tk, err); err != nil {
		t.Fatalf("clone after ClearFaults: %v", err)
	}
}

func TestStorageListContentFilter(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"))
	cl.Node("pve1").AddStorage("local", "dir", fakeapi.WithContentTypes("iso", "vztmpl"))
	cl.Node("pve1").AddStorage("local-lvm", "lvmthin", fakeapi.WithContentTypes("images"))
	cl.Node("pve1").AddStorage("any", "dir")

	c := cl.Client(t)

	list, err := c.Nodes("pve1").Storage().List(context.Background(), &storage.ListOptions{Content: []string{"iso"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	ids := make([]string, 0, len(list))
	for _, s := range list {
		ids = append(ids, s.Storage)
	}

	if !slices.Equal(ids, []string{"any", "local"}) {
		t.Fatalf("iso-capable storages: %v", ids)
	}
}

func TestTasksRunning(t *testing.T) {
	cl := fakeapi.NewCluster(t, fakeapi.WithNodes("pve1"), fakeapi.WithManualTasks())
	cl.Node("pve1").AddVM(100, &qemu.Config{})

	c := cl.Client(t)

	tk, err := c.Nodes("pve1").Qemu().Start(context.Background(), 100, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	running := cl.Tasks().Running()
	if !slices.Equal(running, []string{tk.UPID()}) || strings.Split(running[0], ":")[5] != "qmstart" {
		t.Fatalf("running tasks: %v", running)
	}

	cl.Tasks().Complete(tk.UPID())

	if running := cl.Tasks().Running(); len(running) != 0 {
		t.Fatalf("a completed task is not running: %v", running)
	}
}

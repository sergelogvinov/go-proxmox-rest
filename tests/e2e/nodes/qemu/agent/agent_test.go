//go:build e2e

// Package agent_e2e exercises the per-guest QEMU Guest Agent module
// against a live Proxmox VE cluster.
//
// Every agent command requires a real, running guest with a responsive
// QEMU Guest Agent — a fixture this suite cannot safely create (no VM
// creation module exists yet) or safely act on (exec/file-write/
// suspend/shutdown are all real, guest-visible actions). So every check
// here runs against a syntactically valid but guaranteed-nonexistent
// VMID. Unlike the power actions in nodes/qemu/qemu_test.go, every
// agent command checks the guest exists synchronously
// (PVE::QemuConfig->load_config, "check if VM exists") before doing
// anything else — none of them fork a background task — so every one of
// them is expected to fail outright here, with no async-success special
// case to account for.
package agent_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu/agent"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// nonexistentVMID is a valid VMID per Proxmox's own numeric range but is
// never allocated by this suite.
const nonexistentVMID = 999999999

// TestAgentCommandsAbsent verifies that every guest agent command
// returns an error against a nonexistent guest.
func TestAgentCommandsAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping agent tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ac := client.Nodes(cfg.Node).Qemu().Agent()
	ctx := t.Context()

	e2e.RequireError(t, "ping", ac.Ping(ctx, nonexistentVMID))

	_, err := ac.GetTime(ctx, nonexistentVMID)
	e2e.RequireError(t, "get-time", err)

	_, err = ac.Info(ctx, nonexistentVMID)
	e2e.RequireError(t, "info", err)

	_, err = ac.FSFreezeStatus(ctx, nonexistentVMID)
	e2e.RequireError(t, "fsfreeze-status", err)

	_, err = ac.FSFreezeFreeze(ctx, nonexistentVMID)
	e2e.RequireError(t, "fsfreeze-freeze", err)

	_, err = ac.FSFreezeThaw(ctx, nonexistentVMID)
	e2e.RequireError(t, "fsfreeze-thaw", err)

	_, err = ac.FSTrim(ctx, nonexistentVMID)
	e2e.RequireError(t, "fstrim", err)

	_, err = ac.NetworkGetInterfaces(ctx, nonexistentVMID)
	e2e.RequireError(t, "network-get-interfaces", err)

	_, err = ac.GetVCPUs(ctx, nonexistentVMID)
	e2e.RequireError(t, "get-vcpus", err)

	_, err = ac.GetFSInfo(ctx, nonexistentVMID)
	e2e.RequireError(t, "get-fsinfo", err)

	_, err = ac.GetMemoryBlocks(ctx, nonexistentVMID)
	e2e.RequireError(t, "get-memory-blocks", err)

	_, err = ac.GetMemoryBlockInfo(ctx, nonexistentVMID)
	e2e.RequireError(t, "get-memory-block-info", err)

	e2e.RequireError(t, "suspend-hybrid", ac.SuspendHybrid(ctx, nonexistentVMID))
	e2e.RequireError(t, "suspend-ram", ac.SuspendRAM(ctx, nonexistentVMID))
	e2e.RequireError(t, "suspend-disk", ac.SuspendDisk(ctx, nonexistentVMID))
	e2e.RequireError(t, "shutdown", ac.Shutdown(ctx, nonexistentVMID))

	_, err = ac.GetHostname(ctx, nonexistentVMID)
	e2e.RequireError(t, "get-host-name", err)

	_, err = ac.GetOSInfo(ctx, nonexistentVMID)
	e2e.RequireError(t, "get-osinfo", err)

	_, err = ac.GetUsers(ctx, nonexistentVMID)
	e2e.RequireError(t, "get-users", err)

	_, err = ac.GetTimezone(ctx, nonexistentVMID)
	e2e.RequireError(t, "get-timezone", err)
}

// TestAgentSetUserPassword verifies client-side validation and the
// absent-guest error path.
func TestAgentSetUserPassword(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping agent tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ac := client.Nodes(cfg.Node).Qemu().Agent()
	ctx := t.Context()

	err := ac.SetUserPassword(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "set-user-password with nil options", err)

	err = ac.SetUserPassword(ctx, nonexistentVMID, &agent.SetUserPasswordOptions{})
	e2e.RequireError(t, "set-user-password without username", err)

	err = ac.SetUserPassword(ctx, nonexistentVMID, &agent.SetUserPasswordOptions{
		Username: "e2e", Password: "hunter22",
	})
	e2e.RequireError(t, "set-user-password on absent guest", err)
}

// TestAgentExec verifies client-side validation and the absent-guest
// error path for Exec/ExecStatus.
func TestAgentExec(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping agent tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ac := client.Nodes(cfg.Node).Qemu().Agent()
	ctx := t.Context()

	_, err := ac.Exec(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "exec with nil options", err)

	_, err = ac.Exec(ctx, nonexistentVMID, &agent.ExecOptions{})
	e2e.RequireError(t, "exec without command", err)

	_, err = ac.Exec(ctx, nonexistentVMID, &agent.ExecOptions{Command: []string{"/bin/true"}})
	e2e.RequireError(t, "exec on absent guest", err)

	_, err = ac.ExecStatus(ctx, nonexistentVMID, 1)
	e2e.RequireError(t, "exec-status on absent guest", err)
}

// TestAgentFile verifies client-side validation and the absent-guest
// error path for FileRead/FileWrite.
func TestAgentFile(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping agent tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ac := client.Nodes(cfg.Node).Qemu().Agent()
	ctx := t.Context()

	_, err := ac.FileRead(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "file-read with nil options", err)

	_, err = ac.FileRead(ctx, nonexistentVMID, &agent.FileReadOptions{})
	e2e.RequireError(t, "file-read without file", err)

	_, err = ac.FileRead(ctx, nonexistentVMID, &agent.FileReadOptions{File: "/etc/hostname"})
	e2e.RequireError(t, "file-read on absent guest", err)

	err = ac.FileWrite(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "file-write with nil options", err)

	err = ac.FileWrite(ctx, nonexistentVMID, &agent.FileWriteOptions{})
	e2e.RequireError(t, "file-write without file", err)

	err = ac.FileWrite(ctx, nonexistentVMID, &agent.FileWriteOptions{File: "/tmp/e2e", Content: "e2e"})
	e2e.RequireError(t, "file-write on absent guest", err)
}

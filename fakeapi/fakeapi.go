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

// Package fakeapi is an in-memory, HTTP-level stand-in for a Proxmox VE
// cluster, for exercising this module's own client code (and downstream
// consumers of it) without a live cluster. See docs/fakeapi.md for the
// full design rationale; this package follows it with one implementation
// deviation, noted on vmState/containerState in state.go: guest
// configuration is stored as the same flat wire-parameter map the real
// client sends/receives, not as a live qemu.Config/lxc.Config, since the
// encode/decode helpers for those types are unexported in their packages.
//
// A NewCluster call starts an httptest.Server; Cluster.Client returns an
// ordinary *proxmox.Client wired to it, so every subpackage's
// request/response path runs unmodified against the fake, exactly as it
// would against a real cluster.
package fakeapi

import (
	"net/http/httptest"
	"testing"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
)

// fakeTokenID and fakeTokenSecret are the fixed credentials Cluster.Client
// authenticates with. The fake does not check them (any non-empty
// PVEAPIToken header is accepted); they exist only so the wire traffic
// looks like a normal token-authenticated client.
const (
	fakeTokenID     = "fakeapi@pve!fakeapi"
	fakeTokenSecret = "fakeapi-secret"
)

// FailureMode selects how a failed node behaves for requests scoped to it.
// See Cluster.FailNode.
type FailureMode int

const (
	// FailureUnreachable makes every request scoped to the failed node
	// (any /nodes/{name}/... path) return HTTP 596 with a Proxmox-shaped
	// error body, the status real Proxmox's inter-node API proxy returns
	// when it cannot reach a cluster member. proxmox.IsUnexpected reports
	// true for it; proxmox.IsNotFound does not.
	FailureUnreachable FailureMode = iota

	// FailureConnRefused closes the TCP connection outright instead of
	// writing an HTTP response, so callers see a transport error (no
	// *proxmox.APIError to unwrap) — for exercising code paths that
	// specifically handle network errors distinctly from API errors.
	FailureConnRefused
)

// Cluster is the in-memory Proxmox cluster and its HTTP frontend.
type Cluster struct {
	state  *clusterState
	server *httptest.Server
}

// ClusterOption configures a Cluster at construction time.
type ClusterOption func(*clusterState)

// WithNodes seeds the cluster with the given node names, all online,
// quorate if len(names) > 1 (matching a standalone Proxmox node, which
// reports no cluster-wide quorum entry at all — see Cluster.Status's
// handler in handlers_cluster.go). Proxmox has no concept of a "primary"
// node and neither does the fake. Each node defaults to 4 cores / 8 GiB
// of reported capacity; use Cluster.Node(name).SetResources to change
// that before seeding guests.
func WithNodes(names ...string) ClusterOption {
	return func(s *clusterState) {
		for _, name := range names {
			s.nodes[name] = newNodeState(s, name)
		}
	}
}

// HAGroupOption configures an HA group at seed time, via WithHAGroup.
type HAGroupOption func(*haGroupState)

// WithHAGroupComment sets the group's comment/description, surfaced as
// ha.Group.Comment.
func WithHAGroupComment(comment string) HAGroupOption {
	return func(g *haGroupState) { g.comment = comment }
}

// WithHAGroupNofailback disables automatic failback to a higher-priority
// node once it rejoins the group, surfaced as ha.Group.Nofailback.
func WithHAGroupNofailback() HAGroupOption {
	return func(g *haGroupState) { g.nofailback = true }
}

// WithHAGroupRestricted restricts resources bound to this group to only
// run on the group's nodes, surfaced as ha.Group.Restricted.
func WithHAGroupRestricted() HAGroupOption {
	return func(g *haGroupState) { g.restricted = true }
}

// WithHAGroup seeds an HA group visible via GET /cluster/ha/groups and
// GET/PUT/DELETE /cluster/ha/groups/{group}. nodes is the property-string
// list of member nodes with optional priority, e.g. "node1:2,node2:1"
// (higher number wins), matching ha.Group.Nodes/ha.GroupOptions.Nodes;
// the fake does not require those names to have been passed to WithNodes.
// Unlike WithNodes, groups aren't limited to seed time — a client can
// also Create/Update/Delete them over the wire once the cluster is
// running, exercising cluster/ha's own CRUD surface.
func WithHAGroup(id, nodes string, opts ...HAGroupOption) ClusterOption {
	return func(s *clusterState) {
		g := &haGroupState{id: id, nodes: nodes}
		for _, opt := range opts {
			opt(g)
		}

		s.haGroups[id] = g
	}
}

// WithManualTasks disables the default instant-completion behavior for
// async operations (status actions, config updates via POST, ...): tasks
// stay "running" until a test explicitly completes them via
// Cluster.Tasks().Complete/Fail.
func WithManualTasks() ClusterOption {
	return func(s *clusterState) { s.manual = true }
}

// NewCluster builds a fake Proxmox cluster and starts its httptest.Server.
// If t is non-nil, t.Cleanup closes the server automatically; pass nil
// (legal for an interface-typed parameter, even though testing.TB cannot
// be implemented outside the testing package) when using fakeapi outside
// a test, e.g. from a `go run`-able example, and call Cluster.Close
// yourself.
func NewCluster(t testing.TB, opts ...ClusterOption) *Cluster {
	state := newClusterState()
	for _, opt := range opts {
		if opt != nil {
			opt(state)
		}
	}

	cl := &Cluster{state: state}
	cl.server = httptest.NewServer(newRouter(state))

	if t != nil {
		t.Cleanup(cl.Close)
	}

	return cl
}

// Close shuts down the underlying httptest.Server. Safe to call more than
// once. Not needed when NewCluster was given a non-nil testing.TB, which
// registers this via t.Cleanup already.
func (cl *Cluster) Close() {
	cl.server.Close()
}

// Client returns a *proxmox.Client wired to this cluster over the
// underlying httptest.Server, using a fixed fake API token. Safe to call
// more than once; each call returns an independent *proxmox.Client
// sharing the same backing state, mirroring how multiple real clients can
// point at one cluster. t may be nil (see NewCluster); if non-nil and
// client construction fails (which New only does for options this
// package never passes), the failure is reported via t.Fatalf instead of
// a panic.
func (cl *Cluster) Client(t testing.TB) *proxmox.Client {
	if t != nil {
		t.Helper()
	}

	c, err := proxmox.New(
		proxmox.ClientConfig{},
		proxmox.WithURL(cl.server.URL),
		proxmox.WithTokenAuth(fakeTokenID, fakeTokenSecret),
		proxmox.WithLogger(quietLogger{}),
	)
	if err != nil {
		if t != nil {
			t.Fatalf("fakeapi: building client: %v", err)
		}
		panic(err)
	}

	return c
}

// Node returns a handle for seeding/mutating one node's state. Panics if
// name was not passed to WithNodes — seeding happens at construction;
// nodes are not added dynamically, matching real cluster-join being out
// of scope (docs/fakeapi.md §2 Non-Goals).
func (cl *Cluster) Node(name string) *Node {
	cl.state.mu.Lock()
	defer cl.state.mu.Unlock()

	n, ok := cl.state.nodes[name]
	if !ok {
		panic("fakeapi: node " + name + " was not seeded via WithNodes")
	}

	return &Node{state: cl.state, node: n}
}

// Tasks returns a handle for driving the async task engine under
// WithManualTasks.
func (cl *Cluster) Tasks() *TaskController {
	return &TaskController{state: cl.state}
}

// FailNode simulates a node going down: every request scoped to it fails
// per mode, and its entry in GET /cluster/status and GET
// /cluster/resources flips to offline, recomputing quorum from the
// surviving online count. Guests already running on the failed node keep
// their last-known state (mirroring how a real node's pvestatd simply
// stops reporting rather than guests visibly changing state) — their
// entries in /cluster/resources are left at last-seen values, not
// synthetically zeroed. Panics if name was not passed to WithNodes.
func (cl *Cluster) FailNode(name string, mode FailureMode) {
	cl.state.mu.Lock()
	defer cl.state.mu.Unlock()

	n, ok := cl.state.nodes[name]
	if !ok {
		panic("fakeapi: node " + name + " was not seeded via WithNodes")
	}

	n.online = false
	n.failing = true
	n.failMode = mode
}

// RecoverNode undoes a prior FailNode, making the node reachable and
// online again. Panics if name was not passed to WithNodes.
func (cl *Cluster) RecoverNode(name string) {
	cl.state.mu.Lock()
	defer cl.state.mu.Unlock()

	n, ok := cl.state.nodes[name]
	if !ok {
		panic("fakeapi: node " + name + " was not seeded via WithNodes")
	}

	n.online = true
	n.failing = false
}

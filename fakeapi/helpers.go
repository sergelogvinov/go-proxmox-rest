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

package fakeapi

import (
	"strconv"
	"time"
)

// memoryMBToBytes converts a guest's memory config (stored in MiB, as
// both qemu.Config and lxc.Config express it) to the bytes status/
// resources endpoints report.
func memoryMBToBytes(mb int) int64 {
	return int64(mb) * 1024 * 1024
}

// guestUptime returns how long a running guest has been up, or zero if
// it isn't running or startedAt was never set.
func guestUptime(running bool, startedAt time.Time) time.Duration {
	if !running || startedAt.IsZero() {
		return 0
	}
	return time.Since(startedAt)
}

// fakePID returns a deterministic, non-zero fake process id for a
// running guest, so status responses look realistic without tracking
// real PIDs.
func fakePID(vmid int) int {
	return 10000 + vmid
}

// sizeToBytes parses a Proxmox volume size string: a bare number in
// kilobytes, or a number with a "K"/"M"/"G"/"T" suffix, matching
// storage.CreateVolumeOptions.Size's documented grammar. Returns 0 for an
// unparseable value.
func sizeToBytes(s string) int64 {
	if s == "" {
		return 0
	}

	mult := int64(1024) // bare number is kilobytes
	numeric := s

	switch s[len(s)-1] {
	case 'T', 't':
		mult = 1024 * 1024 * 1024 * 1024
		numeric = s[:len(s)-1]
	case 'G', 'g':
		mult = 1024 * 1024 * 1024
		numeric = s[:len(s)-1]
	case 'M', 'm':
		mult = 1024 * 1024
		numeric = s[:len(s)-1]
	case 'K', 'k':
		mult = 1024
		numeric = s[:len(s)-1]
	}

	n, err := strconv.ParseFloat(numeric, 64)
	if err != nil {
		return 0
	}

	return int64(n * float64(mult))
}

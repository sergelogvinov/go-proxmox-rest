package types

// network.go: IPAddress is identical between nodes/qemu/agent's
// NetworkInterface.IPAddresses (GET
// /nodes/{node}/qemu/{vmid}/agent/network-get-interfaces, backed by the
// QEMU guest agent's guest-network-get-interfaces command) and
// nodes/lxc's Interface.IPAddresses (GET
// /nodes/{node}/lxc/{vmid}/interfaces) — both report the same
// ip-address/ip-address-type/prefix triple per address, one guest-agent
// sourced and the other read directly from the container's network
// namespace.

// IPAddress is a single address bound to a guest network interface.
type IPAddress struct {
	// IPAddressType is "ipv4" or "ipv6".
	IPAddressType string `json:"ip-address-type,omitempty" url:"ip-address-type,omitempty"`
	// IPAddress is the address itself, e.g. "192.0.2.10".
	IPAddress string `json:"ip-address,omitempty" url:"ip-address,omitempty"`
	// Prefix is the address's subnet prefix length.
	Prefix int `json:"prefix,omitempty" url:"prefix,omitempty"`
}

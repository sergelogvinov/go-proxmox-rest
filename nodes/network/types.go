package network

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Type is a network interface's type.
type Type string

const (
	TypeBridge     Type = "bridge"
	TypeBond       Type = "bond"
	TypeEth        Type = "eth"
	TypeAlias      Type = "alias"
	TypeVLAN       Type = "vlan"
	TypeFabric     Type = "fabric"
	TypeOVSBridge  Type = "OVSBridge"
	TypeOVSBond    Type = "OVSBond"
	TypeOVSPort    Type = "OVSPort"
	TypeOVSIntPort Type = "OVSIntPort"
	TypeVnet       Type = "vnet"
	// TypeUnknown is reported by Get/List when Proxmox cannot classify
	// the interface; never valid on Create/Update.
	TypeUnknown Type = "unknown"

	// TypeAnyBridge, TypeAnyLocalBridge and TypeIncludeSDN are List-only
	// pseudo-filters (matching any bridge, any local bridge, or pulling
	// in SDN-managed interfaces respectively) — they narrow List's
	// result and are never an actual interface's Type.
	TypeAnyBridge      Type = "any_bridge"
	TypeAnyLocalBridge Type = "any_local_bridge"
	TypeIncludeSDN     Type = "include_sdn"
)

// Method is an interface's network configuration method, reported
// separately for IPv4 (Interface.Method) and IPv6 (Interface.Method6).
// Read-only: Proxmox derives it from whether Address/Address6 is set.
type Method string

const (
	MethodLoopback Method = "loopback"
	MethodDHCP     Method = "dhcp"
	MethodManual   Method = "manual"
	MethodStatic   Method = "static"
	MethodAuto     Method = "auto"
)

// BondMode is a bonding interface's link-aggregation mode.
type BondMode string

const (
	BondModeBalanceRR      BondMode = "balance-rr"
	BondModeActiveBackup   BondMode = "active-backup" // OVS and Linux
	BondModeBalanceXOR     BondMode = "balance-xor"
	BondModeBroadcast      BondMode = "broadcast"
	BondMode8023AD         BondMode = "802.3ad"
	BondModeBalanceTLB     BondMode = "balance-tlb"
	BondModeBalanceALB     BondMode = "balance-alb"
	BondModeBalanceSLB     BondMode = "balance-slb"      // OVS
	BondModeLACPBalanceSLB BondMode = "lacp-balance-slb" // OVS
	BondModeLACPBalanceTCP BondMode = "lacp-balance-tcp" // OVS
)

// BondXmitHashPolicy selects the transmit hash policy used for slave
// selection in the balance-xor and 802.3ad bonding modes.
type BondXmitHashPolicy string

const (
	BondXmitHashPolicyLayer2     BondXmitHashPolicy = "layer2"
	BondXmitHashPolicyLayer2Plus BondXmitHashPolicy = "layer2+3"
	BondXmitHashPolicyLayer3Plus BondXmitHashPolicy = "layer3+4"
)

// VLANProtocol is a VLAN-aware bridge's tagging protocol.
type VLANProtocol string

const (
	VLANProtocol8021Q  VLANProtocol = "802.1q"
	VLANProtocol8021AD VLANProtocol = "802.1ad"
)

// Interface describes a network interface, as returned by
// GET /nodes/{node}/network and GET /nodes/{node}/network/{iface}.
//
// Get does not populate Iface (Proxmox's network_config handler returns
// the interface's raw config object without the key it's stored under,
// unlike List's index handler, which injects it) — Client.Get fills it in
// from the requested iface itself.
type Interface struct {
	// Iface is the interface name, e.g. "eth0", "vmbr0".
	Iface string `json:"iface,omitempty" url:"iface,omitempty"`
	// Type is the interface type.
	Type Type `json:"type,omitempty" url:"type,omitempty"`
	// Active is true if the interface is currently active.
	Active bool `json:"active,omitempty" url:"active,omitempty"`
	// Exists is true if the interface physically exists.
	Exists bool `json:"exists,omitempty" url:"exists,omitempty"`
	// Autostart is true if the interface starts automatically on boot.
	Autostart bool `json:"autostart,omitempty" url:"autostart,omitempty"`
	// Comments is the IPv4-section comment. Proxmox stores it as
	// "#"-prefixed comment lines and always appends a trailing "\n"
	// when reassembling them (PVE::Network::Interfaces), so a
	// single-line value written via InterfaceOptions.Comments comes
	// back here with one trailing newline.
	Comments string `json:"comments,omitempty" url:"comments,omitempty"`
	// Comments6 is the IPv6-section comment; see Comments for the same
	// trailing-newline behavior.
	Comments6 string `json:"comments6,omitempty" url:"comments6,omitempty"`

	// BridgeVLANAware is true if VLAN-aware bridging is enabled.
	BridgeVLANAware bool `json:"bridge_vlan_aware,omitempty" url:"bridge_vlan_aware,omitempty"`
	// BridgeVIDs is the allowed VLAN list, e.g. "2 4 100-200".
	BridgeVIDs string `json:"bridge_vids,omitempty" url:"bridge_vids,omitempty"`
	// BridgePorts is the space-separated list of interfaces added to
	// this bridge.
	BridgePorts string `json:"bridge_ports,omitempty" url:"bridge_ports,omitempty"`
	// BridgeAccess is the bridge port's access VLAN.
	BridgeAccess int `json:"bridge-access,omitempty" url:"bridge-access,omitempty"`
	// BridgeLearning is the bridge port's learning flag.
	BridgeLearning bool `json:"bridge-learning,omitempty" url:"bridge-learning,omitempty"`
	// BridgeARPNDSuppress is the bridge port's ARP/ND suppress flag.
	BridgeARPNDSuppress bool `json:"bridge-arp-nd-suppress,omitempty" url:"bridge-arp-nd-suppress,omitempty"`
	// BridgeUnicastFlood is the bridge port's unicast flood flag.
	BridgeUnicastFlood bool `json:"bridge-unicast-flood,omitempty" url:"bridge-unicast-flood,omitempty"`
	// BridgeMulticastFlood is the bridge port's multicast flood flag.
	BridgeMulticastFlood bool `json:"bridge-multicast-flood,omitempty" url:"bridge-multicast-flood,omitempty"`

	// OVSPorts is the space-separated list of interfaces added to this
	// OVS bridge.
	OVSPorts string `json:"ovs_ports,omitempty" url:"ovs_ports,omitempty"`
	// OVSTag is the VLAN tag used by OVSPort/OVSIntPort/OVSBond.
	OVSTag int `json:"ovs_tag,omitempty" url:"ovs_tag,omitempty"`
	// OVSOptions is a raw OVS interface options string.
	OVSOptions string `json:"ovs_options,omitempty" url:"ovs_options,omitempty"`
	// OVSBridge is the OVS bridge this OVS port belongs to.
	OVSBridge string `json:"ovs_bridge,omitempty" url:"ovs_bridge,omitempty"`

	// Slaves is the space-separated list of interfaces in this bonding
	// device.
	Slaves string `json:"slaves,omitempty" url:"slaves,omitempty"`
	// OVSBonds is the space-separated list of interfaces in this OVS
	// bonding device.
	OVSBonds string `json:"ovs_bonds,omitempty" url:"ovs_bonds,omitempty"`
	// BondMode is the bonding link-aggregation mode.
	BondMode BondMode `json:"bond_mode,omitempty" url:"bond_mode,omitempty"`
	// BondPrimary is the primary interface for an active-backup bond.
	BondPrimary string `json:"bond-primary,omitempty" url:"bond-primary,omitempty"`
	// BondXmitHashPolicy is the slave-selection hash policy for
	// balance-xor/802.3ad bonds.
	BondXmitHashPolicy BondXmitHashPolicy `json:"bond_xmit_hash_policy,omitempty" url:"bond_xmit_hash_policy,omitempty"`

	// VLANRawDevice is a VLAN interface's underlying raw device.
	VLANRawDevice string `json:"vlan-raw-device,omitempty" url:"vlan-raw-device,omitempty"`
	// VLANID is a custom-named VLAN interface's VLAN id
	// (ifupdown2 only).
	VLANID int `json:"vlan-id,omitempty" url:"vlan-id,omitempty"`
	// VLANProtocol is a VLAN-aware bridge's tagging protocol.
	VLANProtocol VLANProtocol `json:"vlan-protocol,omitempty" url:"vlan-protocol,omitempty"`

	// Gateway is the IPv4 default gateway address.
	Gateway string `json:"gateway,omitempty" url:"gateway,omitempty"`
	// Netmask is the IPv4 network mask.
	Netmask string `json:"netmask,omitempty" url:"netmask,omitempty"`
	// Address is the IPv4 address.
	Address string `json:"address,omitempty" url:"address,omitempty"`
	// Gateway6 is the IPv6 default gateway address.
	Gateway6 string `json:"gateway6,omitempty" url:"gateway6,omitempty"`
	// Netmask6 is the IPv6 prefix length (0-128).
	Netmask6 int `json:"netmask6,omitempty" url:"netmask6,omitempty"`
	// Address6 is the IPv6 address.
	Address6 string `json:"address6,omitempty" url:"address6,omitempty"`
	// MTU is the interface's MTU.
	MTU int `json:"mtu,omitempty" url:"mtu,omitempty"`

	// Method is the resolved IPv4 configuration method.
	Method Method `json:"method,omitempty" url:"method,omitempty"`
	// Method6 is the resolved IPv6 configuration method.
	Method6 Method `json:"method6,omitempty" url:"method6,omitempty"`
	// Families lists the network families configured on this interface
	// ("inet", "inet6").
	Families []string `json:"families,omitempty" url:"families,omitempty"`
	// Options lists additional raw IPv4 interface options.
	Options []string `json:"options,omitempty" url:"options,omitempty"`
	// Options6 lists additional raw IPv6 interface options.
	Options6 []string `json:"options6,omitempty" url:"options6,omitempty"`
	// Altnames lists alternative (legacy/udev) names for this
	// interface, if any.
	Altnames []string `json:"altnames,omitempty" url:"altnames,omitempty"`
	// Priority is the interface's order.
	Priority int `json:"priority,omitempty" url:"priority,omitempty"`
	// LinkType is the underlying link type.
	LinkType string `json:"link-type,omitempty" url:"link-type,omitempty"`
	// UplinkID is the SDN uplink id, when this interface is an SDN
	// uplink.
	UplinkID string `json:"uplink-id,omitempty" url:"uplink-id,omitempty"`

	// VXLANID is the VXLAN ID.
	VXLANID int `json:"vxlan-id,omitempty" url:"vxlan-id,omitempty"`
	// VXLANSvcNodeIP is the VXLAN service node IP.
	VXLANSvcNodeIP string `json:"vxlan-svcnodeip,omitempty" url:"vxlan-svcnodeip,omitempty"`
	// VXLANPhysdev is the physical device used for the VXLAN tunnel.
	VXLANPhysdev string `json:"vxlan-physdev,omitempty" url:"vxlan-physdev,omitempty"`
	// VXLANLocalTunnelIP is the VXLAN local tunnel IP.
	VXLANLocalTunnelIP string `json:"vxlan-local-tunnelip,omitempty" url:"vxlan-local-tunnelip,omitempty"`
}

// InterfaceOptions holds the write parameters shared by
// POST /nodes/{node}/network (Create) and
// PUT /nodes/{node}/network/{iface} (Update).
//
// Type is required by both Create and Update (Proxmox's schema does not
// mark it optional on either, unlike every other field here). Pointer
// fields are always sent when non-nil (even when zero/empty), which is
// how Update expresses "clear this field"; Create simply sends whatever
// is set. Delete is meaningful to Update only — Client.Create strips it
// from the encoded request, since Proxmox's create_network schema does
// not accept it at all.
type InterfaceOptions struct {
	// Type is the network interface type.
	Type Type `url:"type"`
	// Comments is the IPv4-section comment.
	Comments *string `url:"comments"`
	// Comments6 is the IPv6-section comment.
	Comments6 *string `url:"comments6"`
	// Autostart starts the interface automatically on boot.
	Autostart *bool `url:"autostart"`

	// BridgeVLANAware enables VLAN-aware bridging.
	BridgeVLANAware *bool `url:"bridge_vlan_aware"`
	// BridgeVIDs is the allowed VLAN list, e.g. "2 4 100-200". Only
	// used if the bridge is VLAN aware.
	BridgeVIDs *string `url:"bridge_vids"`
	// BridgePorts, OVSPorts, Slaves and OVSBonds are space-separated
	// interface name lists, e.g. "eth0 eth1". Kept as opaque strings
	// rather than []string: Encode's []string convention comma-joins,
	// which would send the wrong separator here.
	BridgePorts *string `url:"bridge_ports"`
	OVSPorts    *string `url:"ovs_ports"`
	Slaves      *string `url:"slaves"`
	OVSBonds    *string `url:"ovs_bonds"`

	// OVSTag is the VLAN tag used by OVSPort/OVSIntPort/OVSBond.
	OVSTag *int `url:"ovs_tag"`
	// OVSOptions is a raw OVS interface options string.
	OVSOptions *string `url:"ovs_options"`
	// OVSBridge is the OVS bridge this OVS port belongs to; required
	// when Type is TypeOVSIntPort or TypeOVSBond.
	OVSBridge *string `url:"ovs_bridge"`

	// BondMode is the bonding link-aggregation mode.
	BondMode *BondMode `url:"bond_mode"`
	// BondPrimary is the primary interface for an active-backup bond.
	BondPrimary *string `url:"bond-primary"`
	// BondXmitHashPolicy is the slave-selection hash policy for
	// balance-xor/802.3ad bonds.
	BondXmitHashPolicy *BondXmitHashPolicy `url:"bond_xmit_hash_policy"`

	// VLANRawDevice is a VLAN interface's underlying raw device.
	VLANRawDevice *string `url:"vlan-raw-device"`
	// VLANID is a custom-named VLAN interface's VLAN id
	// (ifupdown2 only).
	VLANID *int `url:"vlan-id"`

	// Gateway is the IPv4 default gateway address.
	Gateway *string `url:"gateway"`
	// Netmask is the IPv4 network mask; requires Address.
	Netmask *string `url:"netmask"`
	// Address is the IPv4 address; requires Netmask.
	Address *string `url:"address"`
	// CIDR is the IPv4 address+mask as CIDR, e.g. "10.0.0.1/24" — an
	// alternative to setting Address/Netmask separately; conflicts with
	// both.
	CIDR *string `url:"cidr"`
	// MTU is the interface's MTU (1280-65520).
	MTU *int `url:"mtu"`

	// Gateway6 is the IPv6 default gateway address.
	Gateway6 *string `url:"gateway6"`
	// Netmask6 is the IPv6 prefix length (0-128); requires Address6.
	Netmask6 *int `url:"netmask6"`
	// Address6 is the IPv6 address; requires Netmask6.
	Address6 *string `url:"address6"`
	// CIDR6 is the IPv6 address+prefix as CIDR — an alternative to
	// setting Address6/Netmask6 separately; conflicts with both.
	CIDR6 *string `url:"cidr6"`

	// Delete lists properties to reset to their default value. Update
	// only; Proxmox's create_network schema rejects it outright.
	Delete []string `url:"delete,omitempty,writeonly"`
}

// encode converts the options to form parameters.
func (o *InterfaceOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("network: interface options are required")
	}
	if o.Type == "" {
		return nil, fmt.Errorf("network: interface type is required")
	}

	return params.Encode(o)
}

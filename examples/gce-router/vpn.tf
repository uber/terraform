# An example of how to connect two GCE networks with a VPN
provider "google" {
    credentials = "${file("~/gce/nits-account.json")}"
    project = "${var.project}"
    region = "${var.region1}"
}

resource "google_compute_network" "network1" {
    name = "network1"
    mode = "custom"
}

resource "google_compute_subnetwork" "subnet1" {
    name = "subnet1"
    ip_cidr_range = "10.21.0.0/16"
    network = "${google_compute_network.network1.self_link}"
}

resource "google_compute_subnetwork" "subnet2" {
    name = "subnet2"
    ip_cidr_range = "192.168.1.0/24"
    network = "${google_compute_network.network1.self_link}"
}

resource "google_compute_vpn_gateway" "vpn1" {
    name = "vpn1"
    network = "${google_compute_network.network1.self_link}"
}

resource "google_compute_address" "vpn_static_ip1" {
    name = "vpn-static-ip1"
}

resource "google_compute_forwarding_rule" "fr1_esp" {
    name = "fr1-esp"
    region = "${var.region1}"
    ip_protocol = "ESP"
    ip_address = "${google_compute_address.vpn_static_ip1.address}"
    target = "${google_compute_vpn_gateway.vpn1.self_link}"
}

resource "google_compute_forwarding_rule" "fr1_udp500" {
    name = "fr1-udp500"
    ip_protocol = "UDP"
    port_range = "500"
    ip_address = "${google_compute_address.vpn_static_ip1.address}"
    target = "${google_compute_vpn_gateway.vpn1.self_link}"
}

resource "google_compute_forwarding_rule" "fr1_udp4500" {
    name = "fr1-udp4500"
    ip_protocol = "UDP"
    port_range = "4500"
    ip_address = "${google_compute_address.vpn_static_ip1.address}"
    target = "${google_compute_vpn_gateway.vpn1.self_link}"
}

resource "google_compute_cloud_router" "router1" {
    name = "router1"
    network = "${google_compute_network.network1.self_link}"
    bgp {
        asn = "65001"
    }
    
    bgp_peer {
        name = "bgp-peer1" 
        interface_name = "if-bgp-peer1"
        ip_address = "169.254.1.1"
        peer_ip_address = "169.254.1.2"
        peer_asn = "65002"
    }
    
    interface {
        name = "if-bgp-peer1"
        ip_range = "169.254.1.1/30"
        linked_vpn_tunnel = "tunnel1"
    }
}

resource "google_compute_vpn_tunnel" "tunnel1" {
    name = "tunnel1"
    peer_ip = "130.0.0.1"
    shared_secret = "a secret message"
    target_vpn_gateway = "${google_compute_vpn_gateway.vpn1.self_link}"
    local_traffic_selector = ["192.168.0.0/16"]
    depends_on = ["google_compute_forwarding_rule.fr1_udp500",
        "google_compute_forwarding_rule.fr1_udp4500",
        "google_compute_forwarding_rule.fr1_esp"]
}

---
layout: "google"
page_title: "Google: google_compute_network"
sidebar_current: "docs-google-compute-network"
description: |-
  Manages a network within GCE.
---

# google\_compute\_network

Manages a network within GCE.

## Example Usage

```
resource "google_compute_network" "default" {
	name = "test"
	ipv4_range = "10.0.0.0/16"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
  Changing this forces a new resource to be created.

* `ipv4_range` - (Optional) The IPv4 address range that machines in this
  network are assigned to, represented as a CIDR range.
  Changing this forces a new resource to be created.

* `mode` - (Optional, Default=`"legacy"`) Specifies how the network treats subnetworks. If it is
  set to `"legacy"`, then subnetworks cannot be associated with this network.
  If it is set to `"auto"`, then a subnetwork is automatically created for each
  region this network exists in. Lastly, if it is set to `"custom"`, then
  custom subnetworks can be associated with this network. For more information, 
  read [the official API](https://cloud.google.com/compute/docs/subnetworks).
  Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `gateway_ipv4` - The IPv4 address of the gateway.
* `subnetworks` - A list of subnetworks associated with this network.
* `self_link` - Server defined URL for the resource.

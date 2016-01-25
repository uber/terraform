---
layout: "google"
page_title: "Google: google_compute_subnetwork"
sidebar_current: "docs-google-compute-subnetwork"
description: |-
  Manages a subnetwork within GCE.
---

# google\_compute\_subnetwork

Manages a subnetwork within GCE.

## Example Usage

```
resource "google_compute_network" "default" {
	name = "testnetwork"
	mode = "custom"
}

resource "google_compute_subnetwork" "default" {
	name = "testsubnetwork"
	ip_cidr_range = "192.168.0.0/16"
    network = "${google_compute_network.default.self_link}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
  Changing this forces a new resource to be created.

* `description` - (Optional) A description for the resource, required by GCE.
  Changing this forces a new resource to be created.

* `region` - (Optional) The region this subnetwork belongs to. If not
  specified, the project region is used.
  Changing this forces a new resource to be created.

* `network` - (Required) The URL of the network this subnetwork belongs.
  Changing this forces a new resource to be created.

* `ip_cidr_range` - (Required) The range of internal address owned by this
  subnetwork in CIDR format.
  Changing this forces a new resource to be created.


## Attributes Reference

The following attributes are exported:

* `gateway_address` - The address for default routes to reach destination
  addresses outside this subnetwork.
* `self_link` - Server defined URL for the resource.

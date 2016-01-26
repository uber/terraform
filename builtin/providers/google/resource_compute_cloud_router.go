package google

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	//"google.golang.org/api/googleapi"
)

func resourceComputeCloudRouter() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeCloudRouterCreate,
		Read:   resourceComputeCloudRouterRead,
		Delete: resourceComputeCloudRouterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"bgp": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"asn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"bgp_peer": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interface_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"peer_asn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"peer_ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"interface": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"linked_vpn_tunnel": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"ip_range": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createBgpPeer(v interface{}) (*computeAlpha.RouterBgpPeer, error) {
	_bgpPeer := v.(map[string]interface{})

	_name := _bgpPeer["name"].(string)

	bgpPeer := &computeAlpha.RouterBgpPeer{
		Name: _name,
	}

	if _peerAsn, err := strconv.ParseInt(_bgpPeer["peer_asn"].(string), 10, 64); err != nil {
		return nil, fmt.Errorf("bgp_peer.peer_asn must be a 64 bit integer: %s", err)
	} else {
		bgpPeer.PeerAsn = _peerAsn
	}

	if val, ok := _bgpPeer["ip_address"]; ok {
		bgpPeer.IpAddress = val.(string)
	}

	if val, ok := _bgpPeer["interface_name"]; ok {
		bgpPeer.InterfaceName = val.(string)
	}

	if val, ok := _bgpPeer["peer_ip_address"]; ok {
		bgpPeer.PeerIpAddress = val.(string)
	}

	return bgpPeer, nil
}

func createInterface(v interface{}, project, region string) (*computeAlpha.RouterInterface, error) {
	_routerInterface := v.(map[string]interface{})

	_name := _routerInterface["name"].(string)

	routerInterface := &computeAlpha.RouterInterface{
		Name: _name,
	}

	if val, ok := _routerInterface["ip_range"]; ok {
		routerInterface.IpRange = val.(string)
	}

	if val, ok := _routerInterface["linked_vpn_tunnel"]; ok {
		routerInterface.LinkedVpnTunnel = fmt.Sprintf("https://www.googleapis.com/compute/alpha/projects/%s/regions/%s/vpnTunnels/%s", project, region, val.(string))
	}

	return routerInterface, nil
}

func createBgp(v interface{}) (*computeAlpha.RouterBgp, error) {
	_bgp := v.(map[string]interface{})

	bgp := &computeAlpha.RouterBgp{}
	if _asn, err := strconv.ParseInt(_bgp["asn"].(string), 10, 64); err != nil {
		return nil, fmt.Errorf("bgp.asn must be a 64 bit integer: %s", err)
	} else {
		bgp.Asn = _asn
	}

	return bgp, nil
}

func resourceComputeCloudRouterCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	region := getOptionalRegion(d, config)
	network := d.Get("network").(string)

	router := &computeAlpha.Router{
		Name:    name,
		Network: network,
	}

	_bgps := d.Get("bgp").([]interface{})
	if len(_bgps) != 1 {
		return fmt.Errorf("You must supply one bgp.asn value")
	}

	if bgp, err := createBgp(_bgps[0]); err != nil {
		return err
	} else {
		router.Bgp = bgp
	}

	if v, ok := d.GetOk("interface"); ok {
		_routerInterfaces := v.([]interface{})
		router.Interfaces = make([]*computeAlpha.RouterInterface, len(_routerInterfaces))
		for i, v := range _routerInterfaces {
			if routerInterface, err := createInterface(v, config.Project, region); err != nil {
				return err
			} else {
				router.Interfaces[i] = routerInterface
			}
		}
	}

	if v, ok := d.GetOk("bgp_peer"); ok {
		_bgpPeers := v.([]interface{})
		router.BgpPeers = make([]*computeAlpha.RouterBgpPeer, len(_bgpPeers))
		for i, v := range _bgpPeers {
			if peer, err := createBgpPeer(v); err != nil {
				return err
			} else {
				router.BgpPeers[i] = peer
			}
		}
	}

	if v, ok := d.GetOk("description"); ok {
		router.Description = v.(string)
	}

	op, err := config.clientComputeAlpha.Routers.Insert(config.Project, region, router).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to insert Router %s: %s", name, err)
	}

	err = computeAlphaOperationWaitRegion(config, op, region, "Insert Router")

	if err != nil {
		return fmt.Errorf("Error, failed waitng to insert Router %s: %s", name, err)
	}

	return resourceComputeCloudRouterRead(d, meta)
}

func resourceComputeCloudRouterRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	region := getOptionalRegion(d, config)

	router, err := config.clientComputeAlpha.Routers.Get(config.Project, region, name).Do()
	if err != nil {
		return err
	}

	d.Set("self_link", router.SelfLink)
	d.SetId(name)

	return nil
}

func resourceComputeCloudRouterDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	region := getOptionalRegion(d, config)

	op, err := config.clientComputeAlpha.Routers.Delete(config.Project, region, name).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to delete Router %s: %s", name, err)
	}

	err = computeAlphaOperationWaitRegion(config, op, region, "Delete Router")

	if err != nil {
		return fmt.Errorf("Error, failed waitng to delete Router %s: %s", name, err)
	}

	return nil
}

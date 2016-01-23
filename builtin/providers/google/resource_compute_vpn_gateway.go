package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/googleapi"
)

func resourceComputeVpnGateway() *schema.Resource {
	return &schema.Resource{
		// Unfortunately, the VPNGatewayService does not support update
		// operations. This is why everything is marked forcenew
		Create: resourceComputeVpnGatewayCreate,
		Read:   resourceComputeVpnGatewayRead,
		Delete: resourceComputeVpnGatewayDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeVpnGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	network := d.Get("network").(string)
	region := getOptionalRegion(d, config)
	project := config.Project

	vpnGateway := &computeBeta.TargetVpnGateway{
		Name:    name,
		Network: network,
	}

	if v, ok := d.GetOk("description"); ok {
		vpnGateway.Description = v.(string)
	}

	op, err := config.clientComputeBeta.TargetVpnGateways.Insert(project, region, vpnGateway).Do()
	if err != nil {
		return fmt.Errorf("Error Inserting VPN Gateway %s into network %s: %s", name, network, err)
	}

	err = computeBetaOperationWaitRegion(config, op, region, "Inserting VPN Gateway")
	if err != nil {
		return fmt.Errorf("Error Waiting to Insert VPN Gateway %s into network %s: %s", name, network, err)
	}

	return resourceComputeVpnGatewayRead(d, meta)
}

func resourceComputeVpnGatewayRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	region := getOptionalRegion(d, config)
	project := config.Project

	vpnGateway, err := config.clientComputeBeta.TargetVpnGateways.Get(project, region, name).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing VPN Gateway %q because it's gone, %s", d.Get("name").(string), err)
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error Reading VPN Gateway %s: %s", name, err)
	}

	d.SetId(name)
	d.Set("self_link", vpnGateway.SelfLink)

	return nil
}

func resourceComputeVpnGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	region := d.Get("region").(string)
	project := config.Project

	vpnGatewaysService := computeBeta.NewTargetVpnGatewaysService(config.clientComputeBeta)

	op, err := vpnGatewaysService.Delete(project, region, name).Do()
	if err != nil {
		return fmt.Errorf("Error Reading VPN Gateway %s: %s", name, err)
	}

	err = computeBetaOperationWaitRegion(config, op, region, "Deleting VPN Gateway")
	if err != nil {
		return fmt.Errorf("Error Waiting to Delete VPN Gateway %s: %s", name, err)
	}

	return nil
}

package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/googleapi"
)

func resourceComputeSubnetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSubnetworkCreate,
		Read:   resourceComputeSubnetworkRead,
		Delete: resourceComputeSubnetworkDelete,

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

			"ip_cidr_range": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"gateway_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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

func resourceComputeSubnetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the subnetwork parameter
	subnetwork := &compute.Subnetwork{
		Name:        d.Get("name").(string),
		IpCidrRange: d.Get("ip_cidr_range").(string),
		Network:     d.Get("network").(string),
	}

	region := getOptionalRegion(d, config)

	if v, ok := d.GetOk("description"); ok {
		subnetwork.Description = v.(string)
	}

	log.Printf("[DEBUG] Subnetwork insert request: %#v", subnetwork)
	op, err := config.clientComputeBeta.Subnetworks.Insert(
		config.Project, region, subnetwork).Do()
	if err != nil {
		return fmt.Errorf("Error creating subnetwork: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(subnetwork.Name)

	err = computeBetaOperationWaitRegion(config, op, region, "Creating Subnetwork")
	if err != nil {
		return err
	}

	return resourceComputeSubnetworkRead(d, meta)
}

func resourceComputeSubnetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	region := getOptionalRegion(d, config)

	subnetwork, err := config.clientComputeBeta.Subnetworks.Get(
		config.Project, region, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Subnetwork %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading subnetwork: %s", err)
	}

	d.Set("gateway_address", subnetwork.GatewayAddress)
	d.Set("id", subnetwork.Id)
	d.Set("self_link", subnetwork.SelfLink)

	return nil
}

func resourceComputeSubnetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	region := getOptionalRegion(d, config)

	// Delete the subnetwork
	op, err := config.clientComputeBeta.Subnetworks.Delete(
		config.Project, region, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting subnetwork: %s", err)
	}

	err = computeBetaOperationWaitRegion(config, op, region, "Deleting Subnetwork")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

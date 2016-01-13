package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/googleapi"
)

func resourceComputeNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeNetworkCreate,
		Read:   resourceComputeNetworkRead,
		Delete: resourceComputeNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "legacy",
				ForceNew: true,
			},

			"ipv4_range": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"gateway_ipv4": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"subnetworks": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceComputeNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the network parameter
	network := &computeBeta.Network{
		Name: d.Get("name").(string),
	}

	if v, ok := d.GetOk("mode"); ok {
		mode := v.(string)
		if mode == "custom" {
			network.AutoCreateSubnetworks = false
			network.ForceSendFields = []string{"AutoCreateSubnetworks"}
		} else if mode == "auto" {
			network.AutoCreateSubnetworks = true
		} else if mode != "legacy" {
			return fmt.Errorf("Mode must be \"custom\", \"auto\", or \"legacy\" (default)")
		}
	}

	if v, ok := d.GetOk("ipv4_range"); ok {
		network.IPv4Range = v.(string)
	}

	log.Printf("[DEBUG] Network insert request: %#v", network)
	op, err := config.clientComputeBeta.Networks.Insert(
		config.Project, network).Do()
	if err != nil {
		return fmt.Errorf("Error creating network: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(network.Name)

	err = computeBetaOperationWaitGlobal(config, op, "Creating Network")
	if err != nil {
		return err
	}

	return resourceComputeNetworkRead(d, meta)
}

func resourceComputeNetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	network, err := config.clientComputeBeta.Networks.Get(
		config.Project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Network %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading network: %s", err)
	}

	d.Set("gateway_ipv4", network.GatewayIPv4)
	d.Set("self_link", network.SelfLink)

	subnetworks := make([]interface{}, len(network.Subnetworks))

	for i, v := range network.Subnetworks {
		subnetworks[i] = v
	}

	d.Set("subnetworks", subnetworks)

	return nil
}

func resourceComputeNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Delete the network
	op, err := config.clientComputeBeta.Networks.Delete(
		config.Project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting network: %s", err)
	}

	err = computeBetaOperationWaitGlobal(config, op, "Deleting Network")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	computeBeta "google.golang.org/api/compute/v0.beta"
)

func TestAccComputeSubnetwork_basic(t *testing.T) {
	var subnetwork computeBeta.Subnetwork

	networkName := "n" + acctest.RandString(10)
	subnetworkName := "n" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeSubnetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeSubnetwork_basic(networkName, subnetworkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSubnetworkExists(
						"google_compute_subnetwork.foobar", &subnetwork),
				),
			},
		},
	})
}

func testAccCheckComputeSubnetworkDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_network" {
			continue
		}

		_, err := config.clientComputeBeta.Subnetworks.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Subnetwork still exists")
		}
	}

	return nil
}

func testAccCheckComputeSubnetworkExists(n string, network *computeBeta.Subnetwork) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientComputeBeta.Subnetworks.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Subnetwork not found")
		}

		*network = *found

		return nil
	}
}

func testAccComputeSubnetwork_basic(network, subnetwork string) string {
	return fmt.Sprintf(`
	resource "google_compute_network" "foobar" {
		name = "%s"
		mode = "custom"
	}

	resource "google_compute_subnetwork" "foobar" {
		name = "%s"
		network = "${google_compute_network.foobar.self_link}"
		ip_cidr_range = "10.0.0.0/24"
	}
	`, network, subnetwork)
}

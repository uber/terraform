package aws

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
)

func resourceAwsElasticBeanstalkOptionSetting() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"namespace": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsElasticBeanstalkEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticBeanstalkEnvironmentCreate,
		Read:   resourceAwsElasticBeanstalkEnvironmentRead,
		Update: resourceAwsElasticBeanstalkEnvironmentUpdate,
		Delete: resourceAwsElasticBeanstalkEnvironmentDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"application": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"cname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"tier": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					switch value {
					case
						"Worker",
						"WebServer":
						return
					}
					errors = append(errors, fmt.Errorf("%s is not a valid tier. Valid options are WebServer or Worker", value))
					return
				},
				ForceNew: true,
			},
			"setting": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     resourceAwsElasticBeanstalkOptionSetting(),
				Set:      optionSettingValueHash,
			},
			"all_settings": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     resourceAwsElasticBeanstalkOptionSetting(),
				Set:      optionSettingValueHash,
			},
			"solution_stack_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"template_name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"solution_stack_name"},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsElasticBeanstalkEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	// Get values from config
	name := d.Get("name").(string)
	cname := d.Get("cname").(string)
	tier := d.Get("tier").(string)
	app := d.Get("application").(string)
	desc := d.Get("description").(string)
	settings := d.Get("setting").(*schema.Set)
	solutionStack := d.Get("solution_stack_name").(string)
	templateName := d.Get("template_name").(string)

	// TODO set tags
	// Note: at time of writing, you cannot view or edit Tags after creation
	// d.Set("tags", tagsToMap(instance.Tags))
	createOpts := elasticbeanstalk.CreateEnvironmentInput{
		EnvironmentName: aws.String(name),
		ApplicationName: aws.String(app),
		OptionSettings:  extractOptionSettings(settings),
		Tags:            tagsFromMapBeanstalk(d.Get("tags").(map[string]interface{})),
	}

	if desc != "" {
		createOpts.Description = aws.String(desc)
	}

	if cname != "" {
		createOpts.CNAMEPrefix = aws.String(cname)
	}

	if tier != "" {
		var tierType string

		switch tier {
		case "WebServer":
			tierType = "Standard"
		case "Worker":
			tierType = "SQS/HTTP"
		}
		environmentTier := elasticbeanstalk.EnvironmentTier{
			Name: aws.String(tier),
			Type: aws.String(tierType),
		}
		createOpts.Tier = &environmentTier
	}

	if solutionStack != "" {
		createOpts.SolutionStackName = aws.String(solutionStack)
	}

	if templateName != "" {
		createOpts.TemplateName = aws.String(templateName)
	}

	log.Printf("[DEBUG] Elastic Beanstalk Environment create opts: %s", createOpts)
	resp, err := conn.CreateEnvironment(&createOpts)
	if err != nil {
		return err
	}

	// Assign the application name as the resource ID
	d.SetId(*resp.EnvironmentId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Launching", "Updating"},
		Target:     []string{"Ready"},
		Refresh:    environmentStateRefreshFunc(conn, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Elastic Beanstalk Environment (%s) to become ready: %s",
			d.Id(), err)
	}

	return resourceAwsElasticBeanstalkEnvironmentRead(d, meta)
}

func resourceAwsElasticBeanstalkEnvironmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	if d.HasChange("description") {
		if err := resourceAwsElasticBeanstalkEnvironmentDescriptionUpdate(conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("solution_stack_name") {
		if err := resourceAwsElasticBeanstalkEnvironmentSolutionStackUpdate(conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("setting") {
		if err := resourceAwsElasticBeanstalkEnvironmentOptionSettingsUpdate(conn, d); err != nil {
			return err
		}
	}

	return resourceAwsElasticBeanstalkEnvironmentRead(d, meta)
}

func resourceAwsElasticBeanstalkEnvironmentDescriptionUpdate(conn *elasticbeanstalk.ElasticBeanstalk, d *schema.ResourceData) error {
	name := d.Get("name").(string)
	desc := d.Get("description").(string)
	envId := d.Id()

	log.Printf("[DEBUG] Elastic Beanstalk application: %s, update description: %s", name, desc)

	_, err := conn.UpdateEnvironment(&elasticbeanstalk.UpdateEnvironmentInput{
		EnvironmentId: aws.String(envId),
		Description:   aws.String(desc),
	})

	return err
}

func resourceAwsElasticBeanstalkEnvironmentOptionSettingsUpdate(conn *elasticbeanstalk.ElasticBeanstalk, d *schema.ResourceData) error {
	name := d.Get("name").(string)
	envId := d.Id()

	log.Printf("[DEBUG] Elastic Beanstalk application: %s, update options", name)

	req := &elasticbeanstalk.UpdateEnvironmentInput{
		EnvironmentId: aws.String(envId),
	}

	if d.HasChange("setting") {
		o, n := d.GetChange("setting")
		if o == nil {
			o = &schema.Set{F: optionSettingValueHash}
		}
		if n == nil {
			n = &schema.Set{F: optionSettingValueHash}
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		req.OptionSettings = extractOptionSettings(ns.Difference(os))
	}

	if _, err := conn.UpdateEnvironment(req); err != nil {
		return err
	}

	return nil
}

func resourceAwsElasticBeanstalkEnvironmentSolutionStackUpdate(conn *elasticbeanstalk.ElasticBeanstalk, d *schema.ResourceData) error {
	name := d.Get("name").(string)
	solutionStack := d.Get("solution_stack_name").(string)
	envId := d.Id()

	log.Printf("[DEBUG] Elastic Beanstalk application: %s, update solution_stack_name: %s", name, solutionStack)

	_, err := conn.UpdateEnvironment(&elasticbeanstalk.UpdateEnvironmentInput{
		EnvironmentId:     aws.String(envId),
		SolutionStackName: aws.String(solutionStack),
	})

	return err
}

func resourceAwsElasticBeanstalkEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	app := d.Get("application").(string)
	envId := d.Id()

	log.Printf("[DEBUG] Elastic Beanstalk environment read %s: id %s", d.Get("name").(string), d.Id())

	resp, err := conn.DescribeEnvironments(&elasticbeanstalk.DescribeEnvironmentsInput{
		ApplicationName: aws.String(app),
		EnvironmentIds:  []*string{aws.String(envId)},
	})

	if err != nil {
		return err
	}

	if len(resp.Environments) == 0 {
		log.Printf("[DEBUG] Elastic Beanstalk environment properties: could not find environment %s", d.Id())

		d.SetId("")
		return nil
	} else if len(resp.Environments) != 1 {
		return fmt.Errorf("Error reading application properties: found %d environments, expected 1", len(resp.Environments))
	}

	env := resp.Environments[0]

	if *env.Status == "Terminated" {
		log.Printf("[DEBUG] Elastic Beanstalk environment %s was terminated", d.Id())

		d.SetId("")
		return nil
	}

	if err := d.Set("description", env.Description); err != nil {
		return err
	}

	if err := d.Set("cname", env.CNAME); err != nil {
		return err
	}

	return resourceAwsElasticBeanstalkEnvironmentSettingsRead(d, meta)
}

func fetchAwsElasticBeanstalkEnvironmentSettings(d *schema.ResourceData, meta interface{}) (*schema.Set, error) {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	app := d.Get("application").(string)
	name := d.Get("name").(string)

	resp, err := conn.DescribeConfigurationSettings(&elasticbeanstalk.DescribeConfigurationSettingsInput{
		ApplicationName: aws.String(app),
		EnvironmentName: aws.String(name),
	})

	if err != nil {
		return nil, err
	}

	if len(resp.ConfigurationSettings) != 1 {
		return nil, fmt.Errorf("Error reading environment settings: received %d settings groups, expected 1", len(resp.ConfigurationSettings))
	}

	settings := &schema.Set{F: optionSettingValueHash}
	for _, optionSetting := range resp.ConfigurationSettings[0].OptionSettings {
		m := map[string]interface{}{}

		if optionSetting.Namespace != nil {
			m["namespace"] = *optionSetting.Namespace
		} else {
			return nil, fmt.Errorf("Error reading environment settings: option setting with no namespace: %v", optionSetting)
		}

		if optionSetting.OptionName != nil {
			m["name"] = *optionSetting.OptionName
		} else {
			return nil, fmt.Errorf("Error reading environment settings: option setting with no name: %v", optionSetting)
		}

		if optionSetting.Value != nil {
			switch *optionSetting.OptionName {
			case "SecurityGroups":
				m["value"] = dropGeneratedSecurityGroup(*optionSetting.Value, meta)
			case "Subnets", "ELBSubnets":
				m["value"] = sortValues(*optionSetting.Value)
			default:
				m["value"] = *optionSetting.Value
			}
		}

		settings.Add(m)
	}

	return settings, nil
}

func resourceAwsElasticBeanstalkEnvironmentSettingsRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Elastic Beanstalk environment settings read %s: id %s", d.Get("name").(string), d.Id())

	allSettings, err := fetchAwsElasticBeanstalkEnvironmentSettings(d, meta)
	if err != nil {
		return err
	}

	settings := d.Get("setting").(*schema.Set)

	log.Printf("[DEBUG] Elastic Beanstalk allSettings: %s", allSettings.GoString())
	log.Printf("[DEBUG] Elastic Beanstalk settings: %s", settings.GoString())

	// perform the set operation with only name/namespace as keys, excluding value
	// this is so we override things in the settings resource data key with updated values
	// from the api.  we skip values we didn't know about before because there are so many
	// defaults set by the eb api that we would delete many useful defaults.
	//
	// there is likely a better way to do this
	allSettingsKeySet := schema.NewSet(optionSettingKeyHash, allSettings.List())
	settingsKeySet := schema.NewSet(optionSettingKeyHash, settings.List())
	updatedSettingsKeySet := allSettingsKeySet.Intersection(settingsKeySet)

	log.Printf("[DEBUG] Elastic Beanstalk updatedSettingsKeySet: %s", updatedSettingsKeySet.GoString())

	updatedSettings := schema.NewSet(optionSettingValueHash, updatedSettingsKeySet.List())

	log.Printf("[DEBUG] Elastic Beanstalk updatedSettings: %s", updatedSettings.GoString())

	if err := d.Set("all_settings", allSettings.List()); err != nil {
		return err
	}

	if err := d.Set("setting", updatedSettings.List()); err != nil {
		return err
	}

	return nil
}

func resourceAwsElasticBeanstalkEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	opts := elasticbeanstalk.TerminateEnvironmentInput{
		EnvironmentId:      aws.String(d.Id()),
		TerminateResources: aws.Bool(true),
	}

	log.Printf("[DEBUG] Elastic Beanstalk Environment terminate opts: %s", opts)
	_, err := conn.TerminateEnvironment(&opts)

	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Terminating"},
		Target:     []string{"Terminated"},
		Refresh:    environmentStateRefreshFunc(conn, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Elastic Beanstalk Environment (%s) to become terminated: %s",
			d.Id(), err)
	}

	return nil
}

// environmentStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// the creation of the Beanstalk Environment
func environmentStateRefreshFunc(conn *elasticbeanstalk.ElasticBeanstalk, environmentId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeEnvironments(&elasticbeanstalk.DescribeEnvironmentsInput{
			EnvironmentIds: []*string{aws.String(environmentId)},
		})
		if err != nil {
			log.Printf("[Err] Error waiting for Elastic Beanstalk Environment state: %s", err)
			return -1, "failed", fmt.Errorf("[Err] Error waiting for Elastic Beanstalk Environment state: %s", err)
		}

		if resp == nil || len(resp.Environments) == 0 {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		var env *elasticbeanstalk.EnvironmentDescription
		for _, e := range resp.Environments {
			if environmentId == *e.EnvironmentId {
				env = e
			}
		}

		if env == nil {
			return -1, "failed", fmt.Errorf("[Err] Error finding Elastic Beanstalk Environment, environment not found")
		}

		return env, *env.Status, nil
	}
}

// we use the following two functions to allow us to split out defaults
// as they become overridden from within the template
func optionSettingValueHash(v interface{}) int {
	rd := v.(map[string]interface{})
	namespace := rd["namespace"].(string)
	optionName := rd["name"].(string)
	value, _ := rd["value"].(string)
	hk := fmt.Sprintf("%s:%s=%s", namespace, optionName, sortValues(value))
	log.Printf("[DEBUG] Elastic Beanstalk optionSettingValueHash(%#v): %s: hk=%s,hc=%d", v, optionName, hk, hashcode.String(hk))
	return hashcode.String(hk)
}

func optionSettingKeyHash(v interface{}) int {
	rd := v.(map[string]interface{})
	namespace := rd["namespace"].(string)
	optionName := rd["name"].(string)
	hk := fmt.Sprintf("%s:%s", namespace, optionName)
	log.Printf("[DEBUG] Elastic Beanstalk optionSettingKeyHash(%#v): %s: hk=%s,hc=%d", v, optionName, hk, hashcode.String(hk))
	return hashcode.String(hk)
}

func sortValues(v string) string {
	values := strings.Split(v, ",")
	sort.Strings(values)
	return strings.Join(values, ",")
}

func extractOptionSettings(s *schema.Set) []*elasticbeanstalk.ConfigurationOptionSetting {
	settings := []*elasticbeanstalk.ConfigurationOptionSetting{}

	if s != nil {
		for _, setting := range s.List() {
			settings = append(settings, &elasticbeanstalk.ConfigurationOptionSetting{
				Namespace:  aws.String(setting.(map[string]interface{})["namespace"].(string)),
				OptionName: aws.String(setting.(map[string]interface{})["name"].(string)),
				Value:      aws.String(setting.(map[string]interface{})["value"].(string)),
			})
		}
	}

	return settings
}

func dropGeneratedSecurityGroup(settingValue string, meta interface{}) string {
	conn := meta.(*AWSClient).ec2conn

	groups := strings.Split(settingValue, ",")

	resp, err := conn.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice(groups),
	})

	if err != nil {
		log.Printf("[DEBUG] Elastic Beanstalk error describing SecurityGroups: %v", err)
		return settingValue
	}

	var legitGroups []string
	for _, group := range resp.SecurityGroups {
		log.Printf("[DEBUG] Elastic Beanstalk SecurityGroup: %v", *group.GroupName)
		if !strings.HasPrefix(*group.GroupName, "awseb") {
			legitGroups = append(legitGroups, *group.GroupId)
		}
	}

	sort.Strings(legitGroups)

	return strings.Join(legitGroups, ",")
}

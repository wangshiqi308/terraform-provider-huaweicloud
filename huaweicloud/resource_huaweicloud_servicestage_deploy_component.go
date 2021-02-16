package huaweicloud

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/huaweicloud/golangsdk"
	"log"
	"time"
)

func ResourceServiceStageDeployComponentV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceStageV2DeployComponentCreate,
		Read:   resourceServiceStageV2DeployComponentRead,
		Update: resourceServiceStageV2DeployComponentUpdate,
		Delete: resourceServiceStageV2DeployComponentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},

		Schema: map[string]*schema.Schema{ //request and response parameters
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateString64WithChinese,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"application_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"component_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"environment_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"replica": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"flavor_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"version": {
				Type:     schema.TypeString,
				Required: true,
			},
			"configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"env": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										ForceNew: true,
										Required: true,
									},
									"value": {
										Type:     schema.TypeString,
										ForceNew: true,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"artifacts": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"container": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										ForceNew: true,
										Required: true,
									},
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"url": {
										Type:     schema.TypeString,
										Required: true,
									},
									"auth": {
										Type:     schema.TypeString,
										Required: true,
									},
									"storage": {
										Type:     schema.TypeString,
										Required: true,
									},
									"version": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"external_accesses": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"address": {
							Type:     schema.TypeString,
							Required: true,
						},
						"forward_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"refer_resources": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"refer_alias": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"parameters": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceServiceStageV2DeployComponentCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage client: %s", err)
	}
	artifactList := d.Get("artifacts").([]interface{})
	artifacts := make(map[string]Artifacts)
	artifact := Artifacts{}
	if len(artifactList) == 1 {
		artifactMap := artifactList[0].(map[string]interface{})
		containerList := artifactMap["container"].([]interface{})
		if len(containerList) == 1 {
			containerMap := containerList[0].(map[string]interface{})
			artifact.Storage = containerMap["storage"].(string)
			artifact.Auth = containerMap["auth"].(string)
			artifact.Url = containerMap["url"].(string)
			artifact.Type = containerMap["type"].(string)
			artifact.Version = containerMap["version"].(string)
			artifacts[containerMap["name"].(string)] = artifact
		}
	}

	createComponentOpts := CreateDeployComponentOpts{
		Name:             d.Get("name").(string),
		ApplicationID:    d.Get("application_id").(string),
		Description:      d.Get("description").(string),
		ComponentID:      d.Get("component_id").(string),
		EnvironmentID:    d.Get("environment_id").(string),
		FlavorId:         d.Get("flavor_id").(string),
		Replica:          d.Get("replica").(int),
		Version:          d.Get("version").(string),
		Artifact:         artifacts,
		ExternalAccesses: buildExternalAccesses(d),
		ReferResources:   buildReferResources(d),
	}

	configList := d.Get("configuration").([]interface{})
	configs := make(map[string][]Env)
	if len(artifactList) == 1 {
		configMap := configList[0].(map[string]interface{})
		envList := configMap["env"].([]interface{})
		var envs []Env
		for i := 0; i < len(envList); i++ {
			envMap := envList[i].(map[string]interface{})
			env := Env{
				Name:  envMap["name"].(string),
				Value: envMap["value"],
			}
			envs = append(envs, env)
		}
		configs["env"] = envs
	}
	createComponentOpts.Configuration = configs

	n, err := createDeployComponent(ssClient, createComponentOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud Servicestage deploy Component: %s", err)
	}
	d.SetId(n.InstanceID)
	err = d.Set("application_id", createComponentOpts.ApplicationID)
	err = d.Set("component_id", createComponentOpts.ComponentID)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForSSDeployComponentActive(ssClient, createComponentOpts.ApplicationID, createComponentOpts.ComponentID, n.InstanceID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for Servicestage (%s) to become ACTIVE: %s",
			n.InstanceID, stateErr)
	}
	return resourceServiceStageV2DeployComponentRead(d, meta)
}

func resourceServiceStageV2DeployComponentRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))

	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage client: %s", err)
	}
	n, err := getDeployComponent(ssClient, d.Get("application_id").(string), d.Get("component_id").(string), d.Id()).Extract()
	if err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Huaweicloud servicestage deploy Component: %s", err)
	}

	d.Set("job_id", n.JobId)
	return nil
}

func resourceServiceStageV2DeployComponentUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage: %s", err)
	}

	var updateDeployComponentOpts UpdateDeployComponentOpts

	if d.HasChange("name") {
		updateDeployComponentOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		updateDeployComponentOpts.Description = d.Get("description").(string)
	}

	_, err = updateDeployComponent(ssClient, d.Get("application_id").(string), d.Get("component_id").(string), d.Id(), updateDeployComponentOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating Huaweicloud servicestage Deploy Component: %s", err)
	}
	return resourceServiceStageV2DeployComponentRead(d, meta)
}

func resourceServiceStageV2DeployComponentDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForSSDeployComponentDelete(ssClient, d.Get("application_id").(string), d.Get("component_id").(string), d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting Huaweicloud Servicestage deploy component: %s", err)
	}

	d.SetId("")
	return nil
}

func updateDeployComponent(c *golangsdk.ServiceClient, appID, id, instanceID string, opts UpdateDeployComponentOptsBuilder) (r UpdateDeployComponentResult) {
	b, err := opts.ToSSUpdateMap()
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = c.Put(resourceDeployComponentURL(c, appID, id, instanceID), b, &r.Body, &golangsdk.RequestOpts{
		OkCodes: []int{200},
	})
	return
}

func waitForSSDeployComponentDelete(ssClient *golangsdk.ServiceClient, appID, id, instanceID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r, err := getDeployComponent(ssClient, appID, id, instanceID).Extract()
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud deploy component %s", id)
				return r, "DELETED", nil
			}
			return r, "ACTIVE", err
		}

		err = deleteSSDeployComponent(ssClient, appID, id, instanceID).ExtractErr()

		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud deploy component %s", id)
				return r, "DELETED", nil
			}
			if errCode, ok := err.(golangsdk.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					return r, "ACTIVE", nil
				}
			}
			return r, "ACTIVE", err
		}

		return r, "ACTIVE", nil
	}
}

func deleteSSDeployComponent(c *golangsdk.ServiceClient, appID, id, instanceID string) (r DeleteDeployComponentResult) {
	reqOpt := &golangsdk.RequestOpts{OkCodes: []int{200}}
	_, r.Err = c.Delete(resourceDeployComponentURL(c, appID, id, instanceID), reqOpt)
	return
}

func getDeployComponent(c *golangsdk.ServiceClient, appID, id, instanceID string) (r GetDeployComponentResult) {
	_, r.Err = c.Get(resourceDeployComponentURL(c, appID, id, instanceID), &r.Body, nil)
	return
}

func waitForSSDeployComponentActive(ssClient *golangsdk.ServiceClient, appID, id, instanceID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := getDeployComponent(ssClient, appID, id, instanceID).Extract()
		if err != nil {
			return nil, "", err
		}

		return n, "ACTIVE", nil
	}
}

type CreateDeployComponentResult struct {
	commonDeployComponentResult
}

type GetDeployComponentResult struct {
	commonDeployComponentResult
}

type UpdateDeployComponentResult struct {
	commonDeployComponentResult
}

type DeleteDeployComponentResult struct {
	golangsdk.ErrResult
}

type commonDeployComponentResult struct {
	golangsdk.Result
}

type UpdateDeployComponentOpts struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type UpdateDeployComponentOptsBuilder interface {
	ToSSUpdateMap() (map[string]interface{}, error)
}

func (opts CreateDeployComponentOpts) ToSSCreateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(opts, "")
}

func (opts UpdateDeployComponentOpts) ToSSUpdateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(opts, "")
}

func createDeployComponent(c *golangsdk.ServiceClient, opts CreateDeployComponentOpts) (r CreateDeployComponentResult) {
	b, err := opts.ToSSCreateMap()
	if err != nil {
		r.Err = err
		return
	}
	reqOpt := &golangsdk.RequestOpts{OkCodes: []int{200}}
	_, r.Err = c.Post(createDeployComponentURL(c, opts.ApplicationID, opts.ComponentID), b, &r.Body, reqOpt)

	return
}

type ServiceStageDeployComponent struct {
	JobId      string `json:"job_id"`
	InstanceID string `json:"instance_id"`
}

func (r commonDeployComponentResult) Extract() (*ServiceStageDeployComponent, error) {
	s := &ServiceStageDeployComponent{}
	err := r.ExtractInto(&s)
	return s, err
}

func createDeployComponentURL(c *golangsdk.ServiceClient, appID, componentID string) string {
	return c.ServiceURL("cas/applications", appID, "components", componentID, "instances")
}

func resourceDeployComponentURL(c *golangsdk.ServiceClient, appID, id, instanceID string) string {
	return c.ServiceURL("cas/applications", appID, "components", id, "instances", instanceID)
}

func buildExternalAccesses(d *schema.ResourceData) []ExternalAccess {
	var externalAccessList []ExternalAccess

	rawParams := d.Get("external_accesses").([]interface{})
	for i := range rawParams {
		parameter := rawParams[i].(map[string]interface{})
		externalAccess := ExternalAccess{
			Protocol:    parameter["protocol"].(string),
			Address:     parameter["address"].(string),
			ForwardPort: parameter["forward_port"].(int),
		}
		externalAccessList = append(externalAccessList, externalAccess)
	}
	return externalAccessList
}

func buildReferResources(d *schema.ResourceData) []ReferResource {
	var referResourceList []ReferResource

	rawParams := d.Get("refer_resources").([]interface{})
	for i := range rawParams {
		parameter := rawParams[i].(map[string]interface{})
		referResource := ReferResource{
			Id:         parameter["id"].(string),
			Type:       parameter["type"].(string),
			ReferAlias: parameter["refer_alias"].(string),
			Parameters: parameter["parameters"].(map[string]interface{}),
		}
		referResourceList = append(referResourceList, referResource)
	}
	return referResourceList
}

type CreateDeployComponentOpts struct {
	Name             string `json:"name,omitempty"`
	ApplicationID    string
	ComponentID      string
	EnvironmentID    string               `json:"environment_id,omitempty"`
	Description      string               `json:"description,omitempty"`
	FlavorId         string               `json:"flavor_id,omitempty"`
	Replica          int                  `json:"replica,omitempty"`
	Version          string               `json:"version,omitempty"`
	Artifact         map[string]Artifacts `json:"artifacts,omitempty"`
	ExternalAccesses []ExternalAccess     `json:"external_accesses,omitempty"`
	ReferResources   []ReferResource      `json:"refer_resources,omitempty"`
	Configuration    map[string][]Env     `json:"configuration,omitempty"`
}

type Env struct {
	Name  string      `json:"name,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type ExternalAccess struct {
	Protocol    string `json:"protocol,omitempty"`
	Address     string `json:"address,omitempty"`
	ForwardPort int    `json:"forward_port,omitempty"`
}

type ReferResource struct {
	Id         string                 `json:"id,omitempty"`
	Type       string                 `json:"type,omitempty"`
	ReferAlias string                 `json:"refer_alias,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type Artifacts struct {
	Storage string `json:"storage,omitempty"`
	Type    string `json:"type,omitempty"`
	Url     string `json:"url,omitempty"`
	Auth    string `json:"auth,omitempty"`
	Version string `json:"version,omitempty"`
}

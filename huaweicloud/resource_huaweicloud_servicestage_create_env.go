package huaweicloud

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/huaweicloud/golangsdk"
	"log"
	"time"
)

func ResourceServiceStageCreateEnvV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceStageV2EnvCreate,
		Read:   resourceServiceStageV2EnvRead,
		Update: resourceServiceStageV2EnvUpdate,
		Delete: resourceServiceStageV2EnvDelete,
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
			"enterprise_project_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"alias": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"charge_mode": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"base_resources": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"optional_resources": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceServiceStageV2EnvCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage client: %s", err)
	}

	createEnvOpts := CreateEnvOpts{
		Name:              d.Get("name").(string),
		Description:       d.Get("description").(string),
		Vpc:               d.Get("vpc_id").(string),
		BaseResources:     resourceEnvResource("base_resources", d),
		OptionalResources: resourceEnvResource("optional_resources", d),
	}

	epsID := GetEnterpriseProjectID(d, config)
	if epsID != "" {
		createEnvOpts.EnterpriseProjectID = epsID
	}
	n, err := createEnv(ssClient, createEnvOpts).Extract()

	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud Servicestage env: %s", err)
	}
	d.SetId(n.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForSSEnvActive(ssClient, n.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for Servicestage (%s) to become ACTIVE: %s",
			n.ID, stateErr)
	}

	return resourceServiceStageV2EnvRead(d, meta)
}

func resourceServiceStageV2EnvRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))

	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage client: %s", err)
	}
	n, err := getEnv(ssClient, d.Id()).Extract()
	if err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Huaweicloud servicestage env: %s", err)
	}

	d.Set("name", n.Name)

	return nil
}

func resourceServiceStageV2EnvUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage: %s", err)
	}

	var updateEnvOpts UpdateEnvOpts

	if d.HasChange("name") {
		updateEnvOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		updateEnvOpts.Description = d.Get("description").(string)
	}

	_, err = updateEnv(ssClient, d.Id(), updateEnvOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating Huaweicloud servicestage env: %s", err)
	}
	return resourceServiceStageV2EnvRead(d, meta)
}

func resourceServiceStageV2EnvDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForSSEnvDelete(ssClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting Huaweicloud Servicestage env: %s", err)
	}

	d.SetId("")
	return nil
}

type CreateEnvOptsBuilder interface {
	ToSSCreateMap() (map[string]interface{}, error)
}

type UpdateEnvOptsBuilder interface {
	ToSSUpdateMap() (map[string]interface{}, error)
}

type CreateEnvOpts struct {
	Name                string     `json:"name,omitempty"`
	Description         string     `json:"description,omitempty"`
	EnterpriseProjectID string     `json:"enterprise_project_id,omitempty"`
	Vpc                 string     `json:"vpc_id,omitempty"`
	BaseResources       []Resource `json:"base_resources,omitempty"`
	OptionalResources   []Resource `json:"optional_resources,omitempty"`
}

type Resource struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}

type UpdateEnvOpts struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type ServiceStageEnv struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type CreateEnvResult struct {
	commonEnvResult
}

type GetEnvResult struct {
	commonEnvResult
}

type UpdateEnvResult struct {
	commonEnvResult
}

type commonEnvResult struct {
	golangsdk.Result
}

type DeleteEnvResult struct {
	golangsdk.ErrResult
}

func (opts CreateEnvOpts) ToSSCreateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(opts, "")
}

func (opts UpdateEnvOpts) ToSSUpdateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(opts, "")
}

func (r commonEnvResult) Extract() (*ServiceStageEnv, error) {
	s := &ServiceStageEnv{}
	err := r.ExtractInto(&s)
	return s, err
}

func createEnvURL(c *golangsdk.ServiceClient) string {
	return c.ServiceURL("cas/environments")
}

func resourceEnvURL(c *golangsdk.ServiceClient, id string) string {
	return c.ServiceURL("cas/environments", id)
}

func createEnv(c *golangsdk.ServiceClient, opts CreateEnvOptsBuilder) (r CreateEnvResult) {
	b, err := opts.ToSSCreateMap()
	if err != nil {
		r.Err = err
		return
	}
	reqOpt := &golangsdk.RequestOpts{OkCodes: []int{200}}
	_, r.Err = c.Post(createEnvURL(c), b, &r.Body, reqOpt)

	return
}

func updateEnv(c *golangsdk.ServiceClient, id string, opts UpdateEnvOptsBuilder) (r UpdateEnvResult) {
	b, err := opts.ToSSUpdateMap()
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = c.Put(resourceEnvURL(c, id), b, &r.Body, &golangsdk.RequestOpts{
		OkCodes: []int{200},
	})
	return
}

func getEnv(c *golangsdk.ServiceClient, id string) (r GetEnvResult) {
	_, r.Err = c.Get(resourceEnvURL(c, id), &r.Body, nil)
	return
}

func waitForSSEnvActive(ssClient *golangsdk.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := getEnv(ssClient, id).Extract()
		if err != nil {
			return nil, "", err
		}

		return n, "ACTIVE", nil
	}
}

func waitForSSEnvDelete(ssClient *golangsdk.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		r, err := getEnv(ssClient, id).Extract()
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud servicestage Env %s", id)
				return r, "DELETED", nil
			}
			return r, "ACTIVE", err
		}

		err = deleteSSEnv(ssClient, id).ExtractErr()

		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud servicestage Env %s", id)
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

func deleteSSEnv(c *golangsdk.ServiceClient, id string) (r DeleteEnvResult) {
	_, r.Err = c.Delete(resourceEnvURL(c, id), nil)
	return
}

func resourceEnvResource(key string, d *schema.ResourceData) []Resource {
	res := d.Get(key).([]interface{})

	resources := make([]Resource, len(res))
	for i, raw := range res {
		rawMap := raw.(map[string]interface{})
		resources[i] = Resource{
			ID:   rawMap["id"].(string),
			Type: rawMap["type"].(string),
		}
	}
	return resources
}

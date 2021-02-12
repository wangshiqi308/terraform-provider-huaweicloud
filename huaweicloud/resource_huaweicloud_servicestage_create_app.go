package huaweicloud

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/huaweicloud/golangsdk"
	"log"
	"time"
)

func ResourceServiceStageCreateAPPV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceStageV2AppCreate,
		Read:   resourceServiceStageV2AppRead,
		Update: resourceServiceStageV2AppUpdate,
		Delete: resourceServiceStageV2AppDelete,
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
		},
	}
}

func resourceServiceStageV2AppCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage client: %s", err)
	}
	createOpts := CreateAppOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}
	epsID := GetEnterpriseProjectID(d, config)
	if epsID != "" {
		createOpts.EnterpriseProjectID = epsID
	}

	n, err := createApp(ssClient, createOpts).Extract()

	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud Servicestage: %s", err)
	}
	d.SetId(n.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForSSAppActive(ssClient, n.ID),
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

	return resourceServiceStageV2AppRead(d, meta)
}

func waitForSSAppActive(ssClient *golangsdk.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := getApp(ssClient, id).Extract()
		if err != nil {
			return nil, "", err
		}

		return n, "ACTIVE", nil
	}
}

func resourceServiceStageV2AppRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage client: %s", err)
	}

	n, err := getApp(ssClient, d.Id()).Extract()
	if err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Huaweicloud servicestage: %s", err)
	}

	d.Set("name", n.Name)

	return nil
}

func resourceServiceStageV2AppUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage: %s", err)
	}

	var updateOpts UpdateAppOpts

	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		updateOpts.Description = d.Get("description").(string)
	}

	_, err = updateApp(ssClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating Huaweicloud servicestage: %s", err)
	}
	return resourceServiceStageV2AppRead(d, meta)
}

func resourceServiceStageV2AppDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForSSAppDelete(ssClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting Huaweicloud Servicestage: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForSSAppDelete(ssClient *golangsdk.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		r, err := getApp(ssClient, id).Extract()
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud servicestage %s", id)
				return r, "DELETED", nil
			}
			return r, "ACTIVE", err
		}

		err = deleteSSApp(ssClient, id).ExtractErr()

		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud servicestage %s", id)
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

func deleteSSApp(c *golangsdk.ServiceClient, id string) (r DeleteAppResult) {
	_, r.Err = c.Delete(resourceAppURL(c, id), nil)
	return
}

type DeleteAppResult struct {
	golangsdk.ErrResult
}

type GetAppResult struct {
	commonAppResult
}

type UpdateAppResult struct {
	commonAppResult
}

type UpdateAppOptsBuilder interface {
	ToSSUpdateMap() (map[string]interface{}, error)
}

func getApp(c *golangsdk.ServiceClient, id string) (r GetAppResult) {
	_, r.Err = c.Get(resourceAppURL(c, id), &r.Body, nil)
	return
}

func createApp(c *golangsdk.ServiceClient, opts CreateAppOptsBuilder) (r CreateAppResult) {
	b, err := opts.ToSSCreateMap()
	if err != nil {
		r.Err = err
		return
	}
	reqOpt := &golangsdk.RequestOpts{OkCodes: []int{200}}
	_, r.Err = c.Post(createAppURL(c), b, &r.Body, reqOpt)

	return
}

func updateApp(c *golangsdk.ServiceClient, id string, opts UpdateAppOptsBuilder) (r UpdateAppResult) {
	b, err := opts.ToSSUpdateMap()
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = c.Put(resourceAppURL(c, id), b, &r.Body, &golangsdk.RequestOpts{
		OkCodes: []int{200},
	})
	return
}

type UpdateAppOpts struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type CreateAppOpts struct {
	Name                string `json:"name,omitempty"`
	Description         string `json:"description,omitempty"`
	EnterpriseProjectID string `json:"enterprise_project_id,omitempty"`
}

type CreateAppOptsBuilder interface {
	ToSSCreateMap() (map[string]interface{}, error)
}

func (opts CreateAppOpts) ToSSCreateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(opts, "")
}

func (opts UpdateAppOpts) ToSSUpdateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(opts, "")
}

type CreateAppResult struct {
	commonAppResult
}

type commonAppResult struct {
	golangsdk.Result
}

func createAppURL(c *golangsdk.ServiceClient) string {
	return c.ServiceURL("cas/applications")
}

func resourceAppURL(c *golangsdk.ServiceClient, id string) string {
	return c.ServiceURL("cas/applications", id)
}

func (r commonAppResult) Extract() (*ServiceStageApp, error) {
	s := &ServiceStageApp{}
	err := r.ExtractInto(&s)
	return s, err
}

type ServiceStageApp struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

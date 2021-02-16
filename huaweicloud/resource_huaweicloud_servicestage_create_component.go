package huaweicloud

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/huaweicloud/golangsdk"
	"log"
	"time"
)

func ResourceServiceStageCreateComponentV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceStageV2ComponentCreate,
		Read:   resourceServiceStageV2ComponentRead,
		Update: resourceServiceStageV2ComponentUpdate,
		Delete: resourceServiceStageV2ComponentDelete,
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
			"runtime": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"category": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"sub_category": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"build": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"source": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kind": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},
						"spec": {
							Type:     schema.TypeList,
							ForceNew: true,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
									"auth": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
									"repo_ref": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
									"repo_url": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
									"repo_auth": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
									"repo_type": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
									"type": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
									"storage": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceServiceStageV2ComponentCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage client: %s", err)
	}
	b := make(map[string]string)
	for key, val := range d.Get("build").(map[string]interface{}) {
		b[key] = val.(string)
	}

	createComponentOpts := CreateComponentOpts{
		Name:          d.Get("name").(string),
		ApplicationID: d.Get("application_id").(string),
		Description:   d.Get("description").(string),
		Runtime:       d.Get("runtime").(string),
		Category:      d.Get("category").(string),
		SubCategory:   d.Get("sub_category").(string),
		Build:         b,
	}
	source := SourceObject{}
	sourceList := d.Get("source").([]interface{})
	if len(sourceList) == 1 {
		sourceMap := sourceList[0].(map[string]interface{})
		source.Kind = sourceMap["kind"].(string)

		spec := SourceOrArtifact{}

		specList := sourceMap["spec"].([]interface{})
		if len(specList) == 1 {
			specMap := specList[0].(map[string]interface{})
			if source.Kind == "code" {
				spec = SourceOrArtifact{
					RepoAuth: specMap["repo_auth"].(string),
					RepoUrl:  specMap["repo_url"].(string),
					RepoRef:  specMap["repo_ref"].(string),
					RepoType: specMap["repo_type"].(string),
				}
			} else {
				spec = SourceOrArtifact{
					Auth:    sourceMap["auth"].(string),
					Url:     sourceMap["url"].(string),
					Storage: sourceMap["storage"].(string),
					Type:    sourceMap["type"].(string),
				}
			}
		}
		source.Spec = spec
	}
	//createComponentOpts.Source = source
	n, err := createComponent(ssClient, createComponentOpts).Extract()

	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud Servicestage Component: %s", err)
	}
	d.SetId(n.ID)
	err = d.Set("application_id", createComponentOpts.ApplicationID)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForSSComponentActive(ssClient, createComponentOpts.ApplicationID, n.ID),
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

	return resourceServiceStageV2ComponentRead(d, meta)
}

func resourceServiceStageV2ComponentRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))

	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage client: %s", err)
	}
	n, err := getComponent(ssClient, d.Get("application_id").(string), d.Id()).Extract()
	if err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Huaweicloud servicestage Component: %s", err)
	}

	d.Set("name", n.Name)

	return nil
}

func resourceServiceStageV2ComponentUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage: %s", err)
	}

	var updateComponentOpts UpdateComponentOpts

	if d.HasChange("name") {
		updateComponentOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		updateComponentOpts.Description = d.Get("description").(string)
	}

	_, err = updateComponent(ssClient, d.Get("application_id").(string), d.Id(), updateComponentOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating Huaweicloud servicestage Component: %s", err)
	}
	return resourceServiceStageV2ComponentRead(d, meta)
}

func resourceServiceStageV2ComponentDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ssClient, err := config.ServiceStageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud servicestage: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForSSComponentDelete(ssClient, d.Get("application_id").(string), d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting Huaweicloud Servicestage create component: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForSSComponentActive(ssClient *golangsdk.ServiceClient, appID, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := getComponent(ssClient, appID, id).Extract()
		if err != nil {
			return nil, "", err
		}

		if n.Status == 0 {
			return n, "ACTIVE", nil
		}
		if n.Status == 1 {
			return nil, "", fmt.Errorf("component status: '%s'", n.Status)
		}
		return n, "DELETE", nil
	}
}

func waitForSSComponentDelete(ssClient *golangsdk.ServiceClient, appID, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		r, err := getComponent(ssClient, appID, id).Extract()
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud component %s", id)
				return r, "DELETED", nil
			}
			return r, "ACTIVE", err
		}

		err = deleteSSComponent(ssClient, appID, id).ExtractErr()

		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud component %s", id)
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

func getComponent(c *golangsdk.ServiceClient, appID, id string) (r GetComponentResult) {
	_, r.Err = c.Get(resourceComponentURL(c, appID, id), &r.Body, nil)
	return
}

func updateComponent(c *golangsdk.ServiceClient, appID, id string, opts UpdateComponentOptsBuilder) (r UpdateComponentResult) {
	b, err := opts.ToSSUpdateMap()
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = c.Put(resourceComponentURL(c, appID, id), b, &r.Body, &golangsdk.RequestOpts{
		OkCodes: []int{200},
	})
	return
}

type CreateComponentOpts struct {
	Name        string `json:"name,omitempty"`
	Runtime     string `json:"runtime,omitempty"`
	Category    string `json:"category,omitempty"`
	SubCategory string `json:"sub_category,omitempty"`
	Description string `json:"description,omitempty"`
	//Source        SourceObject      `json:"source,omitempty"`
	Build         map[string]string `json:"build,omitempty"`
	ApplicationID string
}

type UpdateComponentOpts struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type SourceObject struct {
	Kind string           `json:"kind,omitempty"`
	Spec SourceOrArtifact `json:"spec,omitempty"`
}

type SourceOrArtifact struct {
	Storage  string `json:"storage,omitempty"`
	Type     string `json:"type,omitempty"`
	Url      string `json:"url,omitempty"`
	Auth     string `json:"auth,omitempty"`
	RepoType string `json:"repo_type,omitempty"`
	RepoUrl  string `json:"repo_url,omitempty"`
	RepoRef  string `json:"repo_ref,omitempty"`
	RepoAuth string `json:"repo_auth,omitempty"`
}

type CreateComponentResult struct {
	commonComponentResult
}

type GetComponentResult struct {
	commonComponentResult
}

type UpdateComponentResult struct {
	commonComponentResult
}

type DeleteComponentResult struct {
	golangsdk.ErrResult
}

type commonComponentResult struct {
	golangsdk.Result
}

type UpdateComponentOptsBuilder interface {
	ToSSUpdateMap() (map[string]interface{}, error)
}

func createComponentURL(c *golangsdk.ServiceClient, appID string) string {
	return c.ServiceURL("cas/applications", appID, "components")
}

func resourceComponentURL(c *golangsdk.ServiceClient, appID string, id string) string {
	return c.ServiceURL("cas/applications", appID, "components", id)
}

func createComponent(c *golangsdk.ServiceClient, opts CreateComponentOpts) (r CreateComponentResult) {
	b, err := opts.ToSSCreateMap()
	if err != nil {
		r.Err = err
		return
	}
	reqOpt := &golangsdk.RequestOpts{OkCodes: []int{200}}
	_, r.Err = c.Post(createComponentURL(c, opts.ApplicationID), b, &r.Body, reqOpt)

	return
}

func (opts CreateComponentOpts) ToSSCreateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(opts, "")
}

func (opts UpdateComponentOpts) ToSSUpdateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(opts, "")
}

func deleteSSComponent(c *golangsdk.ServiceClient, appID, id string) (r DeleteComponentResult) {
	_, r.Err = c.Delete(resourceComponentURL(c, appID, id), nil)
	return
}

func (r commonComponentResult) Extract() (*ServiceStageComponent, error) {
	s := &ServiceStageComponent{}
	err := r.ExtractInto(&s)
	return s, err
}

type ServiceStageComponent struct {
	Name          string       `json:"name"`
	ID            string       `json:"id"`
	Status        int          `json:"status"`
	Runtime       string       `json:"runtime"`
	Category      string       `json:"category"`
	SubCategory   string       `json:"sub_category"`
	Description   string       `json:"description"`
	ProjectID     string       `json:"project_id"`
	ApplicationID string       `json:"application_id"`
	Source        SourceObject `json:"source"`
	//Build         map[string]string `json:"build,omitempty"`
}

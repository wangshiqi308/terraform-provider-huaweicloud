package huaweicloud

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/huaweicloud/golangsdk"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"strings"
	"time"
)

func resourceKubernetesService() *schema.Resource {
	return &schema.Resource{
		Create: resourceCCECreateService,
		Read:   resourceCCEReadService,
		Update: resourceCCEUpdateService,
		Delete: resourceCCEDeleteService,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: resourceKubernetesServiceSchemaV1(),
	}
}

func expandStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = v.(string)
	}
	return result
}

func expandMetadata(in []interface{}) ObjectMeta {
	meta := ObjectMeta{}
	if len(in) < 1 {
		return meta
	}
	m := in[0].(map[string]interface{})

	if v, ok := m["annotations"].(map[string]interface{}); ok && len(v) > 0 {
		meta.Annotations = expandStringMap(m["annotations"].(map[string]interface{}))
	}

	if v, ok := m["labels"].(map[string]interface{}); ok && len(v) > 0 {
		meta.Labels = expandStringMap(m["labels"].(map[string]interface{}))
	}

	if v, ok := m["generate_name"]; ok {
		meta.GenerateName = v.(string)
	}
	if v, ok := m["name"]; ok {
		meta.Name = v.(string)
	}
	if v, ok := m["namespace"]; ok {
		meta.Namespace = v.(string)
	}

	return meta
}

func expandServicePort(l []interface{}) []ServicePort {
	if len(l) == 0 || l[0] == nil {
		return []ServicePort{}
	}
	obj := make([]ServicePort, len(l), len(l))
	for i, n := range l {
		cfg := n.(map[string]interface{})
		obj[i] = ServicePort{
			Port:       int32(cfg["port"].(int)),
			TargetPort: int32(cfg["target_port"].(int)),
		}
		if v, ok := cfg["name"].(string); ok {
			obj[i].Name = v
		}
		if v, ok := cfg["protocol"].(string); ok {
			obj[i].Protocol = Protocol(v)
		}
		if v, ok := cfg["node_port"].(int); ok {
			obj[i].NodePort = int32(v)
		}
	}
	return obj
}

func sliceOfString(slice []interface{}) []string {
	result := make([]string, len(slice), len(slice))
	for i, s := range slice {
		result[i] = s.(string)
	}
	return result
}

func expandServiceSpec(l []interface{}) ServiceSpec {
	if len(l) == 0 || l[0] == nil {
		return ServiceSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := ServiceSpec{}

	if v, ok := in["ports"].([]interface{}); ok && len(v) > 0 {
		obj.Ports = expandServicePort(v)
	}
	if v, ok := in["selector"].(map[string]interface{}); ok && len(v) > 0 {
		obj.Selector = expandStringMap(v)
	}
	if v, ok := in["cluster_ip"].(string); ok {
		obj.ClusterIP = v
	}
	if v, ok := in["type"].(string); ok {
		obj.Type = ServiceType(v)
	}
	if v, ok := in["external_ips"].(*schema.Set); ok && v.Len() > 0 {
		obj.ExternalIPs = sliceOfString(v.List())
	}
	if v, ok := in["session_affinity"].(string); ok {
		obj.SessionAffinity = ServiceAffinity(v)
	}
	if v, ok := in["load_balancer_ip"].(string); ok {
		obj.LoadBalancerIP = v
	}
	if v, ok := in["load_balancer_source_ranges"].(*schema.Set); ok && v.Len() > 0 {
		obj.LoadBalancerSourceRanges = sliceOfString(v.List())
	}
	if v, ok := in["external_name"].(string); ok {
		obj.ExternalName = v
	}
	if v, ok := in["publish_not_ready_addresses"].(bool); ok {
		obj.PublishNotReadyAddresses = v
	}
	if v, ok := in["external_traffic_policy"].(string); ok {
		obj.ExternalTrafficPolicy = ServiceExternalTrafficPolicyType(v)
	}
	if v, ok := in["health_check_node_port"].(int); ok {
		obj.HealthCheckNodePort = int32(v)
	}

	return obj
}

type CreateCCEK8SServiceOptsBuilder interface {
	ToSSCreateMap() (map[string]interface{}, error)
}

type CreateCCEK8SServiceResult struct {
	commonCCEK8SServiceResult
}

type GetCCEServiceResult struct {
	commonCCEK8SServiceResult
}

type commonCCEK8SServiceResult struct {
	golangsdk.Result
}

func (r commonCCEK8SServiceResult) Extract() (*Service, error) {
	s := &Service{}
	err := r.ExtractInto(&s)
	return s, err
}

func (s Service) ToSSCreateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(s, "")
}

func createCCEK8SService(c *golangsdk.ServiceClient, opts CreateCCEK8SServiceOptsBuilder, nameSpace string) (r CreateCCEK8SServiceResult) {
	b, err := opts.ToSSCreateMap()
	if err != nil {
		r.Err = err
		return
	}
	reqOpt := &golangsdk.RequestOpts{OkCodes: []int{201}}
	_, r.Err = c.Post(createCCEK8SServiceURL(c, nameSpace), b, &r.Body, reqOpt)

	return
}

func createCCEK8SServiceURL(c *golangsdk.ServiceClient, nameSpace string) string {
	return c.ServiceURL("namespaces", nameSpace, "services")
}

func getclient(d *schema.ResourceData, config *Config, clasterID string) (*golangsdk.ServiceClient, error) {
	cceClient, err := config.CCEK8SClient(GetRegion(d, config))
	url := cceClient.ResourceBaseURL()
	endPoint := strings.Split(url, "https://")[1]
	cceClient.ResourceBase = fmt.Sprintf("https://%s.%s", clasterID, endPoint)
	return cceClient, err
}

func resourceCCECreateService(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	clasterID := d.Get("cluster_id").(string)
	cceClient, err := getclient(d, config, clasterID)
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud cce k8s client: %s", err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	svc := Service{
		ObjectMeta: metadata,
		Spec:       expandServiceSpec(d.Get("spec").([]interface{})),
	}

	_, err = createCCEK8SService(cceClient, svc, svc.Namespace).Extract()

	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud cce Service: %s", err)
	}
	d.SetId(svc.Name)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForCCEServiceActive(cceClient, svc.Namespace, svc.Name),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for cce Service (%s) to become ACTIVE: %s",
			svc.Name, stateErr)
	}
	return resourceCCEReadService(d, meta)
}

func resourceCCEK8SServiceURL(c *golangsdk.ServiceClient, nameSpace, id string) string {
	return c.ServiceURL("namespaces", nameSpace, "services", id)
}

func getCCEService(c *golangsdk.ServiceClient, nameSpace, id string) (r GetCCEServiceResult) {
	_, r.Err = c.Get(resourceCCEK8SServiceURL(c, nameSpace, id), &r.Body, nil)
	return
}

func waitForCCEServiceActive(ssClient *golangsdk.ServiceClient, nameSpace, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := getCCEService(ssClient, nameSpace, id).Extract()
		if err != nil {
			return nil, "", err
		}

		return n, "ACTIVE", nil
	}
}
func resourceCCEReadService(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	clasterID := d.Get("cluster_id").(string)
	cceClient, err := getclient(d, config, clasterID)
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud cce k8s client: %s", err)
	}
	n, err := getCCEService(cceClient, "default", d.Id()).Extract()
	if err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Huaweicloud cce Service: %s", err)
	}

	d.SetId(n.Name)

	return nil
}

type UpdateCCEServiceOpts struct {
	Name string `json:"name,omitempty"`
}

type UpdateCCEServiceOptsBuilder interface {
	ToSSUpdateMap() (map[string]interface{}, error)
}

func (opts UpdateCCEServiceOpts) ToSSUpdateMap() (map[string]interface{}, error) {
	return golangsdk.BuildRequestBody(opts, "")
}

func updateCCEService(c *golangsdk.ServiceClient, namespace, id string, opts UpdateCCEServiceOptsBuilder) (r UpdateEnvResult) {
	b, err := opts.ToSSUpdateMap()
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = c.Put(resourceCCEK8SServiceURL(c, namespace, id), b, &r.Body, &golangsdk.RequestOpts{
		OkCodes: []int{200},
	})
	return
}

func resourceCCEUpdateService(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	clasterID := d.Get("cluster_id").(string)
	cceClient, err := getclient(d, config, clasterID)
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud cce k8s client: %s", err)
	}

	if d.HasChange("name") {
		var updateOpts UpdateCCEServiceOpts
		updateOpts.Name = d.Id()

		_, err = updateCCEService(cceClient, "default", d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating HuaweiCloud cce Service: %s", err)
		}
	}

	//update tags
	if d.HasChange("tags") {
		computeClient, err := config.ComputeV1Client(GetRegion(d, config))
		if err != nil {
			return fmt.Errorf("Error creating HuaweiCloud compute client: %s", err)
		}

		serverId := d.Get("server_id").(string)
		tagErr := UpdateResourceTags(computeClient, d, "cloudservers", serverId)
		if tagErr != nil {
			return fmt.Errorf("Error updating tags of cce Service %s: %s", d.Id(), tagErr)
		}
	}

	return resourceCCEReadService(d, meta)
}

func waitForCCEServiceDelete(ssClient *golangsdk.ServiceClient, id string) resource.StateRefreshFunc {
	namespace := "default"
	return func() (interface{}, string, error) {
		r, err := getCCEService(ssClient, namespace, id).Extract()
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud cce service Env %s", id)
				return r, "DELETED", nil
			}
			return r, "ACTIVE", err
		}

		err = deleteCCEService(ssClient, namespace, id).ExtractErr()

		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted Huaweicloud cce service Env %s", id)
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

type DeleteCCEServiceResult struct {
	golangsdk.ErrResult
}

func deleteCCEService(c *golangsdk.ServiceClient, namespace, id string) (r DeleteCCEServiceResult) {
	reqOpt := &golangsdk.RequestOpts{OkCodes: []int{200}}
	_, r.Err = c.Delete(resourceCCEK8SServiceURL(c, namespace, id), reqOpt)
	return
}

func resourceCCEDeleteService(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	clasterID := d.Get("cluster_id").(string)
	cceClient, err := getclient(d, config, clasterID)
	if err != nil {
		return fmt.Errorf("Error creating Huaweicloud cce k8s client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForCCEServiceDelete(cceClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting Huaweicloud CCE Service: %s", err)
	}

	d.SetId("")
	return nil
}

func namespacedMetadataSchema(objectName string, generatableName bool) *schema.Schema {
	return namespacedMetadataSchemaIsTemplate(objectName, generatableName, false)
}

func metadataFields(objectName string) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"annotations": {
			Type:        schema.TypeMap,
			Description: fmt.Sprintf("An unstructured key value map stored with the %s that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations", objectName),
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"generation": {
			Type:        schema.TypeInt,
			Description: "A sequence number representing a specific generation of the desired state.",
			Computed:    true,
		},
		"labels": {
			Type:        schema.TypeMap,
			Description: fmt.Sprintf("Map of string keys and values that can be used to organize and categorize (scope and select) the %s. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels", objectName),
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"name": {
			Type:         schema.TypeString,
			Description:  fmt.Sprintf("Name of the %s, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names", objectName),
			Optional:     true,
			ForceNew:     true,
			Computed:     true,
			ValidateFunc: validateName,
		},
		"resource_version": {
			Type:        schema.TypeString,
			Description: fmt.Sprintf("An opaque value that represents the internal version of this %s that can be used by clients to determine when %s has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency", objectName, objectName),
			Computed:    true,
		},
		"self_link": {
			Type:        schema.TypeString,
			Description: fmt.Sprintf("A URL representing this %s.", objectName),
			Computed:    true,
		},
		"uid": {
			Type:        schema.TypeString,
			Description: fmt.Sprintf("The unique in time and space value for this %s. More info: http://kubernetes.io/docs/user-guide/identifiers#uids", objectName),
			Computed:    true,
		},
	}
}

func conditionalDefault(condition bool, defaultValue interface{}) interface{} {
	if !condition {
		return nil
	}

	return defaultValue
}

func namespacedMetadataSchemaIsTemplate(objectName string, generatableName, isTemplate bool) *schema.Schema {
	fields := metadataFields(objectName)
	fields["namespace"] = &schema.Schema{
		Type:        schema.TypeString,
		Description: fmt.Sprintf("Namespace defines the space within which name of the %s must be unique.", objectName),
		Optional:    true,
		ForceNew:    true,
		Default:     conditionalDefault(!isTemplate, "default"),
	}
	if generatableName {
		fields["generate_name"] = &schema.Schema{
			Type:          schema.TypeString,
			Description:   "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
			Optional:      true,
			ForceNew:      true,
			ConflictsWith: []string{"metadata.name"},
		}
		fields["name"].ConflictsWith = []string{"metadata.generate_name"}
	}

	return &schema.Schema{
		Type:        schema.TypeList,
		Description: fmt.Sprintf("Standard %s's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata", objectName),
		Required:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: fields,
		},
	}
}

func resourceKubernetesServiceSchemaV1() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"cluster_id": {
			Type:        schema.TypeString,
			Description: "The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
			Optional:    true,
			ForceNew:    true,
		},
		"kind": {
			Type:        schema.TypeString,
			Description: "The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
			Optional:    true,
			ForceNew:    true,
			Default:     "Service",
		},
		"apiversion": {
			Type:        schema.TypeString,
			Description: "The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
			Optional:    true,
			ForceNew:    true,
			Default:     "v1",
		},
		"metadata": namespacedMetadataSchema("service", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the behavior of a service. https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cluster_ip": {
						Type:        schema.TypeString,
						Description: "The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
						Optional:    true,
						ForceNew:    true,
						Computed:    true,
					},
					"external_ips": {
						Type:        schema.TypeSet,
						Description: "A list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP.  A common example is external load-balancers that are not part of the Kubernetes system.",
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Set:         schema.HashString,
					},
					"external_name": {
						Type:        schema.TypeString,
						Description: "The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires `type` to be `ExternalName`.",
						Optional:    true,
					},
					"external_traffic_policy": {
						Type:         schema.TypeString,
						Description:  "Denotes if this Service desires to route external traffic to node-local or cluster-wide endpoints. `Local` preserves the client source IP and avoids a second hop for LoadBalancer and Nodeport type services, but risks potentially imbalanced traffic spreading. `Cluster` obscures the client source IP and may cause a second hop to another node, but should have good overall load-spreading. More info: https://kubernetes.io/docs/tutorials/services/source-ip/",
						Optional:     true,
						Computed:     true,
						ValidateFunc: validation.StringInSlice([]string{"Local", "Cluster"}, false),
					},
					"load_balancer_ip": {
						Type:        schema.TypeString,
						Description: "Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.",
						Optional:    true,
					},
					"load_balancer_source_ranges": {
						Type:        schema.TypeSet,
						Description: "If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature. More info: http://kubernetes.io/docs/user-guide/services-firewalls",
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Set:         schema.HashString,
					},
					"ports": {
						Type:        schema.TypeList,
						Description: "The list of ports that are exposed by this service. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
						Optional:    true,
						MinItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Description: "The name of this port within the service. All ports within the service must have unique names. Optional if only one ServicePort is defined on this service.",
									Optional:    true,
								},
								"node_port": {
									Type:        schema.TypeInt,
									Description: "The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. More info: http://kubernetes.io/docs/user-guide/services#type--nodeport",
									Computed:    true,
									Optional:    true,
								},
								"port": {
									Type:        schema.TypeInt,
									Description: "The port that will be exposed by this service.",
									Required:    true,
								},
								"protocol": {
									Type:        schema.TypeString,
									Description: "The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.",
									Optional:    true,
									Default:     "TCP",
									ValidateFunc: validation.StringInSlice([]string{
										"TCP",
										"UDP",
										"SCTP",
									}, false),
								},
								"target_port": {
									Type:        schema.TypeInt,
									Description: "Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = \"None\"`. More info: http://kubernetes.io/docs/user-guide/services#defining-a-service",
									Optional:    true,
									Computed:    true,
								},
							},
						},
					},
					"publish_not_ready_addresses": {
						Type:        schema.TypeBool,
						Optional:    true,
						Default:     false,
						Description: "When set to true, indicates that DNS implementations must publish the `notReadyAddresses` of subsets for the Endpoints associated with the Service. The default value is `false`. The primary use case for setting this field is to use a StatefulSet's Headless Service to propagate `SRV` records for its Pods without respect to their readiness for purpose of peer discovery.",
					},
					"selector": {
						Type:        schema.TypeMap,
						Description: "Route service traffic to pods with label keys and values matching this selector. Only applies to types `ClusterIP`, `NodePort`, and `LoadBalancer`. More info: http://kubernetes.io/docs/user-guide/services#overview",
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
					"session_affinity": {
						Type:        schema.TypeString,
						Description: "Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
						Optional:    true,
						Default:     "None",
						ValidateFunc: validation.StringInSlice([]string{
							"ClientIP",
							"None",
						}, false),
					},
					"type": {
						Type:        schema.TypeString,
						Description: "Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. More info: http://kubernetes.io/docs/user-guide/services#overview",
						Optional:    true,
						Default:     "ClusterIP",
						ValidateFunc: validation.StringInSlice([]string{
							"ClusterIP",
							"ExternalName",
							"NodePort",
							"LoadBalancer",
						}, false),
					},
					"health_check_node_port": {
						Type:        schema.TypeInt,
						Description: "Specifies the Healthcheck NodePort for the service. Only effects when type is set to `LoadBalancer` and external_traffic_policy is set to `Local`.",
						Optional:    true,
						Computed:    true,
						ForceNew:    true,
					},
				},
			},
		},
		"wait_for_load_balancer": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Terraform will wait for the load balancer to have at least 1 endpoint before considering the resource created.",
		},
		"status": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"load_balancer": {
						Type:     schema.TypeList,
						Computed: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"ingress": {
									Type:     schema.TypeList,
									Computed: true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"ip": {
												Type:     schema.TypeString,
												Computed: true,
											},
											"hostname": {
												Type:     schema.TypeString,
												Computed: true,
											},
										},
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

type TypeMeta struct {
	Kind       string `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,2,opt,name=apiVersion"`
}

type ObjectMeta struct {
	Name                       string            `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	GenerateName               string            `json:"generateName,omitempty" protobuf:"bytes,2,opt,name=generateName"`
	Namespace                  string            `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	SelfLink                   string            `json:"selfLink,omitempty" protobuf:"bytes,4,opt,name=selfLink"`
	UID                        types.UID         `json:"uid,omitempty" protobuf:"bytes,5,opt,name=uid,casttype=k8s.io/kubernetes/pkg/types.UID"`
	ResourceVersion            string            `json:"resourceVersion,omitempty" protobuf:"bytes,6,opt,name=resourceVersion"`
	Generation                 int64             `json:"generation,omitempty" protobuf:"varint,7,opt,name=generation"`
	DeletionGracePeriodSeconds *int64            `json:"deletionGracePeriodSeconds,omitempty" protobuf:"varint,10,opt,name=deletionGracePeriodSeconds"`
	Labels                     map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
	Annotations                map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`
	Finalizers                 []string          `json:"finalizers,omitempty" patchStrategy:"merge" protobuf:"bytes,14,rep,name=finalizers"`
	ClusterName                string            `json:"clusterName,omitempty" protobuf:"bytes,15,opt,name=clusterName"`
}

type ServicePort struct {
	Name        string   `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Protocol    Protocol `json:"protocol,omitempty" protobuf:"bytes,2,opt,name=protocol,casttype=Protocol"`
	AppProtocol *string  `json:"appProtocol,omitempty" protobuf:"bytes,6,opt,name=appProtocol"`
	Port        int32    `json:"port" protobuf:"varint,3,opt,name=port"`
	TargetPort  int32    `json:"targetPort,omitempty" protobuf:"bytes,4,opt,name=targetPort"`
	NodePort    int32    `json:"nodePort,omitempty" protobuf:"varint,5,opt,name=nodePort"`
}
type IPFamily string
type IPFamilyPolicyType string
type Protocol string
type ServiceExternalTrafficPolicyType string
type ServiceType string
type ServiceAffinity string
type SessionAffinityConfig struct {
	ClientIP *ClientIPConfig `json:"clientIP,omitempty" protobuf:"bytes,1,opt,name=clientIP"`
}

type ClientIPConfig struct {
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,1,opt,name=timeoutSeconds"`
}

type ServiceSpec struct {
	Ports                         []ServicePort                    `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"port" protobuf:"bytes,1,rep,name=ports"`
	Selector                      map[string]string                `json:"selector,omitempty" protobuf:"bytes,2,rep,name=selector"`
	ClusterIP                     string                           `json:"clusterIP,omitempty" protobuf:"bytes,3,opt,name=clusterIP"`
	ClusterIPs                    []string                         `json:"clusterIPs,omitempty" protobuf:"bytes,18,opt,name=clusterIPs"`
	Type                          ServiceType                      `json:"type,omitempty" protobuf:"bytes,4,opt,name=type,casttype=ServiceType"`
	ExternalIPs                   []string                         `json:"externalIPs,omitempty" protobuf:"bytes,5,rep,name=externalIPs"`
	SessionAffinity               ServiceAffinity                  `json:"sessionAffinity,omitempty" protobuf:"bytes,7,opt,name=sessionAffinity,casttype=ServiceAffinity"`
	LoadBalancerIP                string                           `json:"loadBalancerIP,omitempty" protobuf:"bytes,8,opt,name=loadBalancerIP"`
	LoadBalancerSourceRanges      []string                         `json:"loadBalancerSourceRanges,omitempty" protobuf:"bytes,9,opt,name=loadBalancerSourceRanges"`
	ExternalName                  string                           `json:"externalName,omitempty" protobuf:"bytes,10,opt,name=externalName"`
	ExternalTrafficPolicy         ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty" protobuf:"bytes,11,opt,name=externalTrafficPolicy"`
	HealthCheckNodePort           int32                            `json:"healthCheckNodePort,omitempty" protobuf:"bytes,12,opt,name=healthCheckNodePort"`
	PublishNotReadyAddresses      bool                             `json:"publishNotReadyAddresses,omitempty" protobuf:"varint,13,opt,name=publishNotReadyAddresses"`
	SessionAffinityConfig         *SessionAffinityConfig           `json:"sessionAffinityConfig,omitempty" protobuf:"bytes,14,opt,name=sessionAffinityConfig"`
	TopologyKeys                  []string                         `json:"topologyKeys,omitempty" protobuf:"bytes,16,opt,name=topologyKeys"`
	IPFamilies                    []IPFamily                       `json:"ipFamilies,omitempty" protobuf:"bytes,19,opt,name=ipFamilies,casttype=IPFamily"`
	IPFamilyPolicy                *IPFamilyPolicyType              `json:"ipFamilyPolicy,omitempty" protobuf:"bytes,17,opt,name=ipFamilyPolicy,casttype=IPFamilyPolicyType"`
	AllocateLoadBalancerNodePorts *bool                            `json:"allocateLoadBalancerNodePorts,omitempty" protobuf:"bytes,20,opt,name=allocateLoadBalancerNodePorts"`
}

type Service struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec       ServiceSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

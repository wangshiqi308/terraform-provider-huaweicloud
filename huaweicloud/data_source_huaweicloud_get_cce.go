package huaweicloud

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func DataSourceCCEGetCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGetCCEClusterRead,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"usage": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceGetCCEClusterRead(d *schema.ResourceData, meta interface{}) error {
	d.SetId("d734ec4c-70dd-11eb-b817-0255ac10158b")
	return nil
}

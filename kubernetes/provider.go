package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	baseprovider "github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/chnsz/terraform-provider-kubernetes/common"
)

func Provider() *schema.Provider {
	p := baseprovider.Provider()
	schemas := map[string]*schema.Schema{
		"external_headers": {
			Type:     schema.TypeMap,
			Optional: true,
		},
		common.AccessKeyConfiguration: {
			Type:        schema.TypeString,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc(strings.ToUpper(common.AccessKeyConfiguration), ""),
			Description: fmt.Sprintf("Access key for k8s managed in Huawei Cloud CCE. Can be set with %s.", strings.ToUpper(common.AccessKeyConfiguration)),
		},
		common.SecretKeyConfiguration: {
			Type:        schema.TypeString,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc(strings.ToUpper(common.SecretKeyConfiguration), ""),
			Description: fmt.Sprintf("Secret key for k8s managed in Huawei Cloud CCE. Can be set with %s.", strings.ToUpper(common.SecretKeyConfiguration)),
		},
		common.ProjectIdConfiguration: {
			Type:        schema.TypeString,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc(strings.ToUpper(common.ProjectIdConfiguration), ""),
			Description: fmt.Sprintf("Project which contains k8s cluster instance. Can be set with %s.", strings.ToUpper(common.ProjectIdConfiguration)),
		},
		common.SecurityTokenConfiguration: {
			Type:        schema.TypeString,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc(strings.ToUpper(common.SecurityTokenConfiguration), ""),
			Description: fmt.Sprintf("Security token for k8s managed in Huawei Cloud CCE. Can be set with %s.", strings.ToUpper(common.SecurityTokenConfiguration)),
		},
	}
	for k, v := range schemas {
		p.Schema[k] = v
	}

	p.ConfigureContextFunc = buildProviderConfigure(p)
	return p
}

func buildProviderConfigure(p *schema.Provider) schema.ConfigureContextFunc {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		providerCfg, diagErr := baseprovider.ProviderConfigure(ctx, d, p.TerraformVersion)
		if diagErr.HasError() {
			return nil, diagErr
		}

		cfg := providerCfg.GetConfig()
		cfg.UserAgent = fmt.Sprintf("terraform-provider-kubernetes terraform-provider-iac %s", p.TerraformVersion)
		cc := &common.HuaweiCloudCredential{
			AccessKey:     d.Get(common.AccessKeyConfiguration).(string),
			SecretKey:     d.Get(common.SecretKeyConfiguration).(string),
			ProjectId:     d.Get(common.ProjectIdConfiguration).(string),
			SecurityToken: d.Get(common.SecurityTokenConfiguration).(string),
		}

		wt, err := common.BuildWrappers(cc, buildExternalHeaderTransport(d))
		if err != nil {
			return nil, diag.FromErr(err)
		}
		cfg.WrapTransport = wt
		providerCfg.SetConfig(cfg)

		return providerCfg, nil
	}
}

func buildExternalHeaderTransport(d *schema.ResourceData) *common.ExternalHeaderTransport {
	headers := make(map[string]string)
	if extHeaders, ok := d.Get("external_headers").(map[string]any); ok {
		for k, v := range extHeaders {
			if ev, ok := v.(string); ok {
				headers[k] = ev
			}
		}
	}
	if len(headers) == 0 {
		return nil
	}

	return common.NewExternalHeaderTransport(headers)
}

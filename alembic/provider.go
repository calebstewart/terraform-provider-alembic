package alembic

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var stderr = os.Stderr

func New() tfsdk.Provider {
	return &provider{}
}

type provider struct {
	configured   bool
	project_root string
	alembic      []string
}

// GetSchema
func (p *provider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"project_root": {
				Type:        types.StringType,
				Description: "Path to the project root directory where your alembic configuration is stored.",
				Optional:    false,
				Computed:    true,
			},
			"alembic": {
				Type:        types.ListType{ElemType: types.StringType},
				Description: "An argument list which is used as the Alembic command line (default: ['alembic'])",
				Optional:    true,
				Computed:    true,
			},
		},
	}, nil
}

// Provider schema struct
type providerData struct {
	ProjectRoot types.String `tfsdk:"project_root"`
	Alembic     types.List   `tfsdk:"alembic"`
}

func (p *provider) Configure(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
	// Retrieve provider data from configuration
	var config providerData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure the given file path is a directory
	if pathinfo, err := os.Stat(config.ProjectRoot.Value); os.IsNotExist(err) || !pathinfo.IsDir() {
		resp.Diagnostics.AddError("project_root must be an valid directory path", err.Error())
		return
	}

	// Ensure that the alembic configuration exists
	if pathinfo, err := os.Stat(filepath.Join(config.ProjectRoot.Value, "alembic.ini")); os.IsNotExist(err) || pathinfo.IsDir() {
		resp.Diagnostics.AddError("project_root must contain an alembic.ini configuration", err.Error())
		return
	}

	// Everything looks good!
	p.project_root = config.ProjectRoot.Value

	// Optionally configurea custom alembic command
	if !config.Alembic.Unknown && !config.Alembic.Null {
		p.alembic = make([]string, len(config.Alembic.Elems))
		for idx, v := range config.Alembic.Elems {
			p.alembic[idx] = v.String()
		}
	} else {
		p.alembic = []string{"alembic"}
	}

	p.configured = true
}

// GetResources - Defines provider resources
func (p *provider) GetResources(_ context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"alembic_upgrade": resourceUpgradeType{},
		// "alembic_downgrade": resourceDowngradeType{},
		// "alembic_stamp":     resourceStampType{},
	}, nil
}

// GetDataSources - Defines provider data sources
func (p *provider) GetDataSources(_ context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		// "alembic_revision": dataRevisionType{},
	}, nil
}

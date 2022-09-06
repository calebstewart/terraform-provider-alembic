package alembic

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ provider.Provider = &alembicProvider{}

type alembicProvider struct {
	configured   bool
	version      string
	project_root string
	alembic      []string
	config       string
	section      string
	extra        map[string]string
}

// Provider schema struct
type providerData struct {
	ProjectRoot types.String `tfsdk:"project_root"`
	Alembic     types.List   `tfsdk:"alembic"`
	Config      types.String `tfsdk:"config"`
	Section     types.String `tfsdk:"section"`
	Extra       types.Map    `tfsdk:"extra"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &alembicProvider{
			version: version,
		}
	}
}

// GetSchema
func (p *alembicProvider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Alembic database migration integration for Terraform.",
		Attributes: map[string]tfsdk.Attribute{
			"project_root": {
				Type:        types.StringType,
				Description: "Path to the project root directory where your alembic configuration is stored.",
				Required:    true,
			},
			"alembic": {
				Type:        types.ListType{ElemType: types.StringType},
				Description: "An argument list which is used as the Alembic command line (default: ['alembic'])",
				Optional:    true,
			},
			"config": {
				Type:        types.StringType,
				Description: "Name of the alembic configuration file (default: 'alembic.ini')",
				Optional:    true,
			},
			"section": {
				Type:        types.StringType,
				Description: "The section within the configuration file to use for Alembic config (default: 'alembic')",
				Optional:    true,
			},
			"extra": {
				Type:        types.MapType{ElemType: types.StringType},
				Description: "Additional arguments consumed by custom env.py scripts",
				Optional:    true,
			},
		},
	}, nil
}

func (p *alembicProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config providerData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.Config.Null {
		p.config = config.Config.Value
	} else {
		p.config = "alembic.ini"
	}

	if !config.Section.Null {
		p.section = config.Section.Value
	} else {
		p.section = "alembic"
	}

	// Ensure the given file path is a directory
	if pathinfo, err := os.Stat(config.ProjectRoot.Value); os.IsNotExist(err) || !pathinfo.IsDir() {
		resp.Diagnostics.AddError("project_root must be an valid directory path", err.Error())
		return
	}

	// Ensure that the alembic configuration exists
	if pathinfo, err := os.Stat(filepath.Join(config.ProjectRoot.Value, p.config)); os.IsNotExist(err) || pathinfo.IsDir() {
		resp.Diagnostics.AddError("project_root must contain an alembic.ini configuration", err.Error())
		return
	}

	// Everything looks good!
	p.project_root = config.ProjectRoot.Value

	// Optionally configurea custom alembic command
	if !config.Alembic.Unknown && !config.Alembic.Null {
		config.Alembic.ElementsAs(ctx, &p.alembic, false)
	} else {
		p.alembic = []string{"alembic"}
	}

	if !config.Extra.Null {
		resp.Diagnostics.Append(config.Extra.ElementsAs(ctx, &p.extra, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		p.extra = nil
	}

	p.configured = true
}

// GetResources - Defines provider resources
func (p *alembicProvider) GetResources(_ context.Context) (map[string]provider.ResourceType, diag.Diagnostics) {
	return map[string]provider.ResourceType{
		"alembic_upgrade": resourceUpgradeType{},
		"alembic_stamp":   resourceStampType{},
	}, nil
}

// GetDataSources - Defines provider data sources
func (p *alembicProvider) GetDataSources(_ context.Context) (map[string]provider.DataSourceType, diag.Diagnostics) {
	return map[string]provider.DataSourceType{
		// "alembic_revision": dataRevisionType{},
	}, nil
}

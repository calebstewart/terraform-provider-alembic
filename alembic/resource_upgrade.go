package alembic

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceUpgradeType struct{}

func (r resourceUpgradeType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"revision": {
				Type:        types.StringType,
				Description: "Revision identifier. The target revision to which we will upgrade.",
				Required:    true,
			},
			"tag": {
				Type:        types.StringType,
				Description: "Arbitrary 'tag' name - can be used by custom env.py scripts.",
				Optional:    true,
			},
			"environment": {
				Type:        types.MapType{ElemType: types.StringType},
				Description: "Environment variables to set when running the alembic command.",
				Optional:    true,
				Sensitive:   true,
			},
			"alembic": {
				Type:        types.ListType{ElemType: types.StringType},
				Description: "Command used to execute alembic. By default, this is taken from the provider configuration.",
				Optional:    true,
			},
			"upgraded_revision": {
				Type:        types.StringType,
				Description: "The resulting revision after applying the upgrade.",
				Computed:    true,
			},
			"proxy_command": {
				Type:        types.ListType{ElemType: types.StringType},
				Description: "An argument list used to execute a proxy which allows direct communication with the database (e.g. cloud-sql-proxy)",
				Optional:    true,
			},
			"proxy_sleep": {
				Type:        types.StringType,
				Description: "Amount of time to sleep in order to allow the proxy to startup. Format is '[0-9]+(s|m|h|d|M|Y)' (default: '5s')",
				Optional:    true,
				Validators: []tfsdk.AttributeValidator{
					stringvalidator.RegexMatches(durationRegex, "proxy_sleep must be in the format '[0-9]+(s|m|h|d|M|Y)'"),
				},
			},
			"extra": {
				Type:        types.MapType{ElemType: types.StringType},
				Description: "Additional arguments consumed by custom env.py scripts",
				Optional:    true,
			},
		},
	}, nil
}

func (r resourceUpgradeType) NewResource(_ context.Context, p provider.Provider) (resource.Resource, diag.Diagnostics) {
	return resourceUpgrade{
		p: *(p.(*alembicProvider)),
	}, nil
}

type resourceUpgrade struct {
	p alembicProvider
}

type resourceUpgradeData struct {
	Environment      types.Map    `tfsdk:"environment"`
	Alembic          types.List   `tfsdk:"alembic"`
	ProxyCommand     types.List   `tfsdk:"proxy_command"`
	ProxySleep       types.String `tfsdk:"proxy_sleep"`
	UpgradedRevision types.String `tfsdk:"upgraded_revision"`
	Revision         string       `tfsdk:"revision"`
	Extra            types.Map    `tfsdk:"extra"`
	Tag              types.String `tfsdk:"tag"`
}

// Create a new resource
func (r resourceUpgrade) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan resourceUpgradeData

	// Retrieve the plan arguments
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	proxy, diags := executeProxyCommand(ctx, plan.ProxyCommand, plan.ProxySleep)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	defer proxy.Process.Kill()

	diags = r.doCreateOrUpgrade(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Store our updated resourceUpgradeData in the state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	return
}

// Update resource
func (r resourceUpgrade) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceUpgradeData

	// Retrieve the plan arguments
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	proxy, diags := executeProxyCommand(ctx, plan.ProxyCommand, plan.ProxySleep)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	defer proxy.Process.Kill()

	diags = r.doCreateOrUpgrade(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Store our updated resourceUpgradeData in the state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	return
}

func (r resourceUpgrade) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var plan resourceUpgradeData

	// Retrieve the plan arguments
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	upgraded_rev, real_head, diags := doReadState(ctx, r.p, plan.ProxyCommand, plan.ProxySleep, plan.Alembic, plan.Extra, plan.Environment, plan.Revision)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Store the resulting revision ID
	plan.UpgradedRevision.Value = upgraded_rev
	plan.UpgradedRevision.Unknown = false

	if real_head != upgraded_rev {
		plan.Revision = real_head
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	return
}

// Delete resource
func (r resourceUpgrade) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("unable to delete alembic versions", "delete makes no sense for database migrations")
}

// Import resource
func (r resourceUpgrade) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Save the import identifier in the id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// For alembic migrations, upgrade and create look the same, so we abstract that here.
// This function will modify the provided plan object to include the results and return
// a diagnostics instance with any found errors.
func (r resourceUpgrade) doCreateOrUpgrade(ctx context.Context, plan *resourceUpgradeData) diag.Diagnostics {

	var result diag.Diagnostics
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	// Build the base alembic command
	proc, diags := buildUpgradeOrDowngradeCommand(ctx, r.p, plan.Alembic, plan.Extra, plan.Environment, plan.Tag, plan.Revision, "upgrade")
	result.Append(diags...)
	if result.HasError() {
		return result
	}

	// Capture standard error output
	proc.Stdout = &stdout
	proc.Stderr = &stderr

	// Execute alembic
	err := proc.Run()
	if err != nil {
		result.AddError(
			fmt.Sprintf("alembic upgrade failed: %v", err),
			fmt.Sprintf("Standard Output:\n%v\n\nStandard Error:\n%v\n\n", stdout.String(), stderr.String()),
		)
		return result
	}

	// Run alembic again to get the output information for out state file
	proc, diags = buildAlembicCommand(ctx, r.p, plan.Alembic, plan.Extra, plan.Environment, "current")
	result.Append(diags...)
	if result.HasError() {
		return result
	}

	// Reset the stderr output in case alembic wrote to it previously
	stdout.Reset()
	stderr.Reset()

	// Store standard output which has our new revision
	proc.Stdout = &stdout
	proc.Stderr = &stderr

	err = proc.Run()
	if err != nil {
		result.AddError(
			fmt.Sprintf("alembic current failed: %v", err),
			fmt.Sprintf("Standard Output:\n%v\n\nStandard Error:\n%v\n\n", stdout.String(), stderr.String()),
		)
		return result
	}

	// Store the resulting revision ID
	plan.UpgradedRevision.Value = strings.Split(strings.Trim(stdout.String(), "\n\r"), " ")[0]
	plan.UpgradedRevision.Unknown = false

	return result
}

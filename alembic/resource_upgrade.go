package alembic

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
		},
	}, nil
}

func (r resourceUpgradeType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceUpgrade{
		p: *(p.(*provider)),
	}, nil
}

type resourceUpgrade struct {
	p provider
}

type resourceUpgradeData struct {
	Revision         string       `tfsdk:"revision"`
	Tag              types.String `tfsdk:"tag"`
	Environment      types.Map    `tfsdk:"environment"`
	Alembic          types.List   `tfsdk:"alembic"`
	UpgradedRevision types.String `tfsdk:"upgraded_revision"`
}

// Create a new resource
func (r resourceUpgrade) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {

	var plan resourceUpgradeData

	// Retrieve the plan arguments
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

// Read resource information
func (r resourceUpgrade) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {

	var plan resourceUpgradeData
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	// Retrieve the plan arguments
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Run the "alembic current" command to get the current head
	proc, diags := r.buildAlembicCommand(ctx, plan, "current")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reset the stderr output in case alembic wrote to it previously
	stderr.Reset()

	// Store standard output which has our new revision
	proc.Stdout = &stdout
	proc.Stderr = &stderr

	err := proc.Run()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("alembic current failed: %v", err), stderr.String())
		return
	}

	// Store the resulting revision ID
	plan.UpgradedRevision.Value = strings.Trim(stdout.String(), "\n\r")

	// Store our updated resourceUpgradeData in the state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	return
}

// Update resource
func (r resourceUpgrade) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var plan resourceUpgradeData

	// Retrieve the plan arguments
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

// Delete resource
func (r resourceUpgrade) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	resp.Diagnostics.AddWarning("unable to delete alembic versions", "delete makes no sense for database migrations")
}

// Import resource
func (r resourceUpgrade) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	// Save the import identifier in the id attribute
	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// For alembic migrations, upgrade and create look the same, so we abstract that here.
// This function will modify the provided plan object to include the results and return
// a diagnostics instance with any found errors.
func (r resourceUpgrade) doCreateOrUpgrade(ctx context.Context, plan *resourceUpgradeData) diag.Diagnostics {

	var result diag.Diagnostics
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	// Build the base alembic command
	proc, diags := r.buildAlembicCommand(ctx, *plan, "upgrade")
	result.Append(diags...)
	if result.HasError() {
		return result
	}

	// Capture standard error output
	proc.Stderr = &stderr

	// Execute alembic
	err := proc.Run()
	if err != nil {
		result.AddError(fmt.Sprintf("alembic upgrade failed: %v", err), stderr.String())
		return result
	}

	// Run alembic again to get the output information for out state file
	proc, diags = r.buildAlembicCommand(ctx, *plan, "current")
	result.Append(diags...)
	if result.HasError() {
		return result
	}

	// Reset the stderr output in case alembic wrote to it previously
	stderr.Reset()

	// Store standard output which has our new revision
	proc.Stdout = &stdout
	proc.Stderr = &stderr

	err = proc.Run()
	if err != nil {
		result.AddError(fmt.Sprintf("alembic current failed: %v", err), stderr.String())
		return result
	}

	// Store the resulting revision ID
	plan.UpgradedRevision.Value = strings.Trim(stdout.String(), "\n\r")

	return result
}

// Build the alembic Cmd structure from the alembic argument
func (r resourceUpgrade) buildAlembicCommand(ctx context.Context, plan resourceUpgradeData, command string) (*exec.Cmd, diag.Diagnostics) {
	var alembic []string
	var diags diag.Diagnostics

	if !plan.Alembic.Null {
		// Use the alembic command specified in the resource
		alembic = make([]string, len(plan.Alembic.Elems))
		for i, v := range plan.Alembic.Elems {
			alembic[i] = v.String()
		}
	} else {
		// Default to the provider alembic command
		alembic = r.p.alembic
	}

	if command == "upgrade" || command == "downgrade" {
		// Add the plan argument
		if !plan.Tag.Null {
			alembic = append(alembic, "--tag", plan.Tag.Value)
		}

		// Add the revision
		alembic = append(alembic, plan.Revision)
	}

	// Build the command instance
	proc := exec.Command(alembic[0], alembic...)

	// Add environment
	if !plan.Environment.Null {
		var environment map[string]string

		// Retrieve the parsed environment mapping
		diags.Append(plan.Environment.ElementsAs(ctx, environment, false)...)
		if diags.HasError() {
			return nil, diags
		}

		// Add the environment variables to the current environment
		for k, v := range environment {
			proc.Env = append(proc.Env, k+"="+v)
		}
	}

	// Ensure the process runs from the project root directory
	proc.Dir = r.p.project_root

	return proc, diags
}

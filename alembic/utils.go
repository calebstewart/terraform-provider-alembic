package alembic

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var durationRegex = regexp.MustCompile(`P([\d\.]+Y)?([\d\.]+M)?([\d\.]+D)?T?([\d\.]+H)?([\d\.]+M)?([\d\.]+?S)?`)

// ParseDuration converts a ISO8601 duration into a time.Duration
func parseDuration(str string) time.Duration {
	matches := durationRegex.FindStringSubmatch(str)

	years := parseDurationPart(matches[1], time.Hour*24*365)
	months := parseDurationPart(matches[2], time.Hour*24*30)
	days := parseDurationPart(matches[3], time.Hour*24)
	hours := parseDurationPart(matches[4], time.Hour)
	minutes := parseDurationPart(matches[5], time.Second*60)
	seconds := parseDurationPart(matches[6], time.Second)

	return time.Duration(years + months + days + hours + minutes + seconds)
}

func parseDurationPart(value string, unit time.Duration) time.Duration {
	if len(value) != 0 {
		if parsed, err := strconv.ParseFloat(value[:len(value)-1], 64); err == nil {
			return time.Duration(float64(unit) * parsed)
		}
	}
	return 0
}

func executeProxyCommand(ctx context.Context, proxy_command types.List, proxy_sleep types.String) (*exec.Cmd, diag.Diagnostics) {
	var diags diag.Diagnostics
	var args []string
	var sleep_duration time.Duration

	// This is fine, we don't need a proxy command
	if proxy_command.Null {
		return nil, diags
	}

	diags.Append(proxy_command.ElementsAs(ctx, &args, false)...)
	if diags.HasError() {
		return nil, diags
	}

	if !proxy_sleep.Null {
		sleep_duration = parseDuration(proxy_sleep.Value)
	} else {
		sleep_duration = 5 * time.Second
	}

	proc := exec.CommandContext(ctx, args[0], args[1:]...)

	// No stdio for the proxy
	proc.Stdin = nil
	proc.Stderr = nil
	proc.Stdout = nil

	err := proc.Start()
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to start sql proxy: %v", args), err.Error())
		return nil, diags
	}

	// Wait a bit for the proxy to come alive
	time.Sleep(sleep_duration)

	return proc, diags

}

func doReadState(
	ctx context.Context,
	p alembicProvider,
	proxy_command types.List,
	proxy_sleep types.String,
	alembic_command types.List,
	extra_values types.Map,
	environment_values types.Map,
	revision string,
) (string, string, diag.Diagnostics) {

	var diags diag.Diagnostics
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	var upgraded_revision string
	var real_head string

	proxy, result_diags := executeProxyCommand(ctx, proxy_command, proxy_sleep)
	diags.Append(result_diags...)
	if diags.HasError() {
		return "", "", diags
	}
	defer proxy.Process.Kill()

	// Run the "alembic current" command to get the current revision for the database
	proc, result_diags := buildAlembicCommand(ctx, p, alembic_command, extra_values, environment_values, "current")
	diags.Append(result_diags...)
	if diags.HasError() {
		return "", "", diags
	}

	// Store standard output which has our new revision
	proc.Stdout = &stdout
	proc.Stderr = &stderr

	err := proc.Run()
	if err != nil {
		diags.AddError(
			fmt.Sprintf("alembic current failed: %v", err),
			fmt.Sprintf("Standard Output:\n%v\n\nStandard Error:\n%v\n\n", stdout.String(), stderr.String()),
		)
		return "", "", diags
	}

	// Store the resulting revision ID
	upgraded_revision = strings.Split(strings.Trim(stdout.String(), "\n\r"), " ")[0]
	real_head = upgraded_revision

	// If the target revision is the special name "head", we need to resolve this to
	// the revision ID to make sure we are actually at "head", and trigger an update if
	// needed.
	if revision == "head" {
		// the  'alembic show' command can reoslve symbolic names to revision IDs
		proc, result_diags = buildAlembicCommand(ctx, p, alembic_command, extra_values, environment_values, "show", revision)
		diags.Append(result_diags...)
		if diags.HasError() {
			return "", "", diags
		}

		// Capture output
		stderr.Reset()
		stdout.Reset()
		proc.Stderr = &stderr
		proc.Stdout = &stdout

		// Run the process
		err := proc.Run()
		if err != nil {
			diags.AddError(
				fmt.Sprintf("alembic current failed: %v", err),
				fmt.Sprintf("Standard Output:\n%v\n\nStandard Error:\n%v\n\n", stdout.String(), stderr.String()),
			)
			return "", "", diags
		}

		// The first line of the output shoud loook like "Rev: 774ddff6187f (head)"
		expression := `(?m)^Rev: ([a-f0-9]+) \(` + regexp.QuoteMeta(revision) + `\)$`
		exp, err := regexp.Compile(expression)
		if err != nil {
			diags.AddError(fmt.Sprintf("failed to compile revision expression: '%v'", expression), err.Error())
			return "", "", diags
		}

		// Find the matching sub expressions
		matches := exp.FindStringSubmatch(stdout.String())
		if matches == nil || matches[1] == "" {
			diags.AddError(
				"failed parsing alembic show results",
				fmt.Sprintf(
					"The regular expression '%v' did not match. If Alembic changes output formats, please submit an\nissue at https://github.com/calebstewart/terraform-provider-alembic.\n\nStandard Output:\n%v\n",
					expression,
					stdout.String(),
				),
			)
			return "", "", diags
		}

		// The meaning of 'head' changed, so change the state value for revision
		// to match the current head. This will trigger a upgrade for the resource.
		if matches[1] != upgraded_revision {
			real_head = matches[1]
		}
	}

	return upgraded_revision, real_head, diags

}

func buildUpgradeOrDowngradeCommand(
	ctx context.Context,
	p alembicProvider,
	alembic_command types.List,
	extra_values types.Map,
	environment_values types.Map,
	tag types.String,
	revision string,
	args ...string,
) (*exec.Cmd, diag.Diagnostics) {

	// Add the tag if specified
	if !tag.Null {
		args = append(args, "--tag", tag.Value)
	}

	// Add the revision
	args = append(args, revision)

	return buildAlembicCommand(ctx, p, alembic_command, extra_values, environment_values, args...)
}

func buildAlembicCommand(
	ctx context.Context,
	p alembicProvider,
	alembic_command types.List,
	extra_values types.Map,
	environment_values types.Map,
	args ...string,
) (*exec.Cmd, diag.Diagnostics) {

	var alembic []string
	var diags diag.Diagnostics

	if !alembic_command.Null {
		diags.Append(alembic_command.ElementsAs(ctx, &alembic, false)...)
		if diags.HasError() {
			return nil, diags
		}
	} else {
		// Default to the provider alembic command
		alembic = p.alembic
	}

	// Add provider extras
	if p.extra != nil {
		for k, v := range p.extra {
			alembic = append(alembic, "-x", fmt.Sprintf("%v=%v", k, v))
		}
	}

	// Add resource extras
	if !extra_values.Null {
		var extra map[string]string
		diags.Append(extra_values.ElementsAs(ctx, &extra, false)...)
		if diags.HasError() {
			return nil, diags
		}

		for k, v := range extra {
			alembic = append(alembic, "-x", fmt.Sprintf("%v=%v", k, v))
		}
	}

	// Add our specific alembic sub-command
	alembic = append(alembic, args...)

	// Build the command instance
	proc := exec.Command(alembic[0], alembic[1:]...)

	// We should only need the PATH variable to locate binaries
	proc.Env = []string{fmt.Sprintf("PATH=%v", os.Getenv("PATH"))}

	// Add custom environment
	if !environment_values.Null {
		var environment map[string]string

		// Retrieve the parsed environment mapping
		diags.Append(environment_values.ElementsAs(ctx, &environment, false)...)
		if diags.HasError() {
			return nil, diags
		}

		// Add the environment variables to the current environment
		for k, v := range environment {
			proc.Env = append(proc.Env, k+"="+v)
		}
	}

	// Ensure the process runs from the project root directory
	proc.Dir = p.project_root

	return proc, diags
}

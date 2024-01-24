package common

import (
	"context"
	"fmt"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

type ExecTriggerType string

const (
	ON_CHANGE ExecTriggerType = "on_change"
	ON_START  ExecTriggerType = "on_start"
	ONCE      ExecTriggerType = "once"
)

func (t ExecTriggerType) String() string {
	return string(t)
}

type ExecModel struct {
	Command      types.List   `tfsdk:"command"`
	Environment  types.Map    `tfsdk:"environment"`
	WorkingDir   types.String `tfsdk:"working_dir"`
	Trigger      types.String `tfsdk:"trigger"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	RecordOutput types.Bool   `tfsdk:"record_output"`
	FailOnError  types.Bool   `tfsdk:"fail_on_error"`
	UserID       types.Int64  `tfsdk:"uid"`
	GroupID      types.Int64  `tfsdk:"gid"`
	ExitCode     types.Int64  `tfsdk:"exit_code"`
	Output       types.String `tfsdk:"stdout"`
	Error        types.String `tfsdk:"stderr"`
	RunCount     types.Int64  `tfsdk:"run_count"`
}

// IsTriggered determines whether the exec command needs to be executed.
func (e ExecModel) IsTriggered(isInstanceStarted bool) bool {
	// Disabled exec cannot be triggered.
	if !e.Enabled.ValueBool() {
		return false
	}

	trigger := ExecTriggerType(e.Trigger.ValueString())

	switch trigger {
	case ON_CHANGE:
		return true
	case ON_START:
		return isInstanceStarted
	case ONCE:
		return e.RunCount.ValueInt64() == 0
	default:
		// Unknown trigger type.
		return false
	}
}

// Execute executes the exec command and populates the computed fields,
// such as exit code, stdout, and stderr.
func (e *ExecModel) Execute(ctx context.Context, server lxd.InstanceServer, instanceName string) diag.Diagnostics {
	var diags diag.Diagnostics

	cmd := make([]string, 0, len(e.Command.Elements()))
	env := make(map[string]string, len(e.Environment.Elements()))

	diags.Append(e.Command.ElementsAs(ctx, &cmd, false)...)
	diags.Append(e.Environment.ElementsAs(ctx, &env, false)...)
	if diags.HasError() {
		return diags
	}

	execReq := api.InstanceExecPost{
		Command:      cmd,
		Environment:  env,
		WaitForWS:    true,
		Interactive:  false,
		RecordOutput: false,
		Cwd:          e.WorkingDir.ValueString(),
		User:         uint32(e.GroupID.ValueInt64()),
		Group:        uint32(e.UserID.ValueInt64()),
	}

	// Create buffers to capture stdout and stderr.
	var outBuf utils.Buffer
	var errBuf utils.Buffer

	if e.RecordOutput.ValueBool() {
		outBuf = utils.NewBufferCloser()
		errBuf = utils.NewBufferCloser()
	} else {
		outBuf = utils.NewDiscardCloser()
		errBuf = utils.NewDiscardCloser()
	}

	execArgs := lxd.InstanceExecArgs{
		Stdout:   outBuf,
		Stderr:   errBuf,
		DataDone: make(chan bool),
	}

	// Exit code -1 indicates the command was not executed.
	exitCode := int64(-1)

	// Run command.
	opExec, err := server.ExecInstance(instanceName, execReq, &execArgs)
	if err == nil {
		err = opExec.WaitContext(ctx)
		if err == nil {
			// Wait for any remaining output to be flushed.
			select {
			case <-ctx.Done():
				err = ctx.Err()
			case <-execArgs.DataDone:
			}
		}

		// Extract exit code from operation's metadata.
		opMeta := opExec.Get().Metadata
		if opMeta != nil {
			rc, ok := opMeta["return"].(float64)
			if ok {
				exitCode = int64(rc)
			}
		}
	}

	// Fail on error (only if user requested).
	if e.FailOnError.ValueBool() && (err != nil || exitCode != 0) {
		diags.AddError(
			fmt.Sprintf("Failed to execute command on instance %q", instanceName),
			fmt.Sprintf("Command %q failed with an error (%d): %v", strings.Join(cmd, " "), exitCode, err),
		)
		return diags
	}

	// Set command's computed values.
	e.RunCount = types.Int64Value(e.RunCount.ValueInt64() + 1)
	e.ExitCode = types.Int64Value(exitCode)
	e.Output = types.StringValue(outBuf.String())
	e.Error = types.StringValue(errBuf.String())

	if e.RecordOutput.ValueBool() && err != nil {
		// If output is recorded and error is not nil, set
		// error as stderr, because errBuf will be empty.
		e.Error = types.StringValue(err.Error())
	}

	return nil
}

// ToExecMap converts execs schema into map of exec models.
func ToExecMap(ctx context.Context, execMap types.Map) (map[string]*ExecModel, diag.Diagnostics) {
	if execMap.IsNull() || execMap.IsUnknown() {
		return nil, nil
	}

	execs := make(map[string]*ExecModel, len(execMap.Elements()))
	diags := execMap.ElementsAs(ctx, &execs, false)

	// Set default computed values (if needed).
	for k, e := range execs {
		e.ExitCode = types.Int64Value(-1)
		e.Output = types.StringValue("")
		e.Error = types.StringValue("")

		if e.RunCount.IsUnknown() {
			e.RunCount = types.Int64Value(0)
		}

		execs[k] = e
	}

	return execs, diags
}

// ToExecMapType converts map of exec models into schema type.
func ToExecMapType(ctx context.Context, execs map[string]*ExecModel) (types.Map, diag.Diagnostics) {
	execType := map[string]attr.Type{
		"command":       types.ListType{ElemType: types.StringType},
		"environment":   types.MapType{ElemType: types.StringType},
		"working_dir":   types.StringType,
		"trigger":       types.StringType,
		"enabled":       types.BoolType,
		"record_output": types.BoolType,
		"fail_on_error": types.BoolType,
		"uid":           types.Int64Type,
		"gid":           types.Int64Type,
		"exit_code":     types.Int64Type,
		"stdout":        types.StringType,
		"stderr":        types.StringType,
		"run_count":     types.Int64Type,
	}

	return types.MapValueFrom(ctx, types.ObjectType{AttrTypes: execType}, execs)
}

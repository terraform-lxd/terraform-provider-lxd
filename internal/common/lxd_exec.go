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

type ExecModel struct {
	Command      types.List   `tfsdk:"command"`
	Triggers     types.List   `tfsdk:"triggers"`
	Environment  types.Map    `tfsdk:"environment"`
	WorkingDir   types.String `tfsdk:"working_dir"`
	RecordOutput types.Bool   `tfsdk:"record_output"`
	FailOnError  types.Bool   `tfsdk:"fail_on_error"`
	UserID       types.Int64  `tfsdk:"uid"`
	GroupID      types.Int64  `tfsdk:"gid"`
	ExitCode     types.Int64  `tfsdk:"exit_code"`
	Output       types.String `tfsdk:"stdout"`
	Error        types.String `tfsdk:"stderr"`
}

func (e *ExecModel) Execute(ctx context.Context, server lxd.InstanceServer, instanceName string) diag.Diagnostics {
	var diags diag.Diagnostics

	env := make(map[string]string, len(e.Environment.Elements()))
	command := make([]string, 0, len(e.Command.Elements()))

	diags.Append(e.Environment.ElementsAs(ctx, &env, false)...)
	diags.Append(e.Command.ElementsAs(ctx, &command, false)...)
	if diags.HasError() {
		return diags
	}

	// If command is one liner, split it on spaces.
	if len(command) == 1 {
		command = strings.SplitN(command[0], " ", 3)
	}

	execReq := api.InstanceExecPost{
		Command:      command,
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

	execArgs := &lxd.InstanceExecArgs{
		Stdout: outBuf,
		Stderr: errBuf,
	}

	// Run command.
	opExec, err := server.ExecInstance(instanceName, execReq, execArgs)
	if err == nil {
		err = opExec.WaitContext(ctx)
	}

	// Extract exit code from operation's metadata.
	var exitCode int64
	opMeta := opExec.Get().Metadata
	if opMeta != nil {
		rc, ok := opMeta["return"].(float64)
		if ok {
			exitCode = int64(rc)
		}
	}

	e.ExitCode = types.Int64Value(exitCode)

	// Fail on error (only if user requested).
	if e.FailOnError.ValueBool() && (err != nil || exitCode != 0) {
		diags.AddError(
			fmt.Sprintf("Failed to execute command on instance %q", instanceName),
			fmt.Sprintf("Command %q failed with an error: %v", strings.Join(command, " "), err),
		)
		return diags
	}

	// Store command output.
	e.Output = types.StringValue(outBuf.String())
	e.Error = types.StringValue(errBuf.String())

	if e.RecordOutput.ValueBool() && err != nil {
		// If output is recorded and error is not nil, set
		// error as stderr, because errBuf will be empty.
		e.Error = types.StringValue(err.Error())
	}

	return nil
}

// Equals compares two execs and determines whether they are equal.
func (e1 ExecModel) Equal(e2 ExecModel) bool {
	if !e1.Command.Equal(e2.Command) {
		return false
	}

	if !e1.Triggers.Equal(e2.Triggers) {
		return false
	}

	return true
}

// ExecSlicesEqual returns true if there is no new command to be
// executed.
func ExecSlicesEqual(old []ExecModel, new []ExecModel) bool {
	// If new exec block were added, new commands need
	// to be run.
	if len(new) > len(old) {
		return false
	}

	// If any exec block has been changed, new commands
	// need to be run.
	for i := range old {
		if i >= len(new) {
			break
		}

		if !old[i].Equal(new[i]) {
			return false
		}
	}

	return true
}

// ToExecList converts list of exec blocks of type types.List
// into list of exec structures.
func ToExecList(ctx context.Context, execList types.List) ([]ExecModel, diag.Diagnostics) {
	if execList.IsNull() {
		return []ExecModel{}, nil
	}

	execs := make([]ExecModel, 0, len(execList.Elements()))
	diags := execList.ElementsAs(ctx, &execs, false)
	return execs, diags
}

// ToExecList converts list of exec blocks of type types.List
// into list of exec structures.
func ToExecListType(ctx context.Context, execs []ExecModel) (types.List, diag.Diagnostics) {
	execType := map[string]attr.Type{
		"command":       types.ListType{ElemType: types.StringType},
		"triggers":      types.ListType{ElemType: types.StringType},
		"environment":   types.MapType{ElemType: types.StringType},
		"working_dir":   types.StringType,
		"record_output": types.BoolType,
		"fail_on_error": types.BoolType,
		"uid":           types.Int64Type,
		"gid":           types.Int64Type,
		"exit_code":     types.Int64Type,
		"stdout":        types.StringType,
		"stderr":        types.StringType,
	}

	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: execType}, execs)
}

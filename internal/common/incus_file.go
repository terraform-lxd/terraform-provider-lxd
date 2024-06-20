package common

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lxc/incus/v6/client"
	"github.com/mitchellh/go-homedir"

	"github.com/lxc/terraform-provider-incus/internal/errors"
)

type InstanceFileModel struct {
	Content    types.String `tfsdk:"content"`
	SourcePath types.String `tfsdk:"source_path"`
	TargetPath types.String `tfsdk:"target_path"`
	UserID     types.Int64  `tfsdk:"uid"`
	GroupID    types.Int64  `tfsdk:"gid"`
	Mode       types.String `tfsdk:"mode"`
	CreateDirs types.Bool   `tfsdk:"create_directories"`
	Append     types.Bool   `tfsdk:"append"`
}

// ToFileMap converts files from types.Set into map[string]IncusFileModel.
func ToFileMap(ctx context.Context, fileSet types.Set) (map[string]InstanceFileModel, diag.Diagnostics) {
	if fileSet.IsNull() || fileSet.IsUnknown() {
		return make(map[string]InstanceFileModel), nil
	}

	files := make([]InstanceFileModel, 0, len(fileSet.Elements()))
	diags := fileSet.ElementsAs(ctx, &files, false)
	if diags.HasError() {
		return nil, diags
	}

	// Convert list into map.
	fileMap := make(map[string]InstanceFileModel, len(files))
	for _, f := range files {
		fileMap[f.TargetPath.ValueString()] = f
	}

	return fileMap, diags
}

// ToFileSetType converts files from a map[string]IncusFileModel into types.Set.
func ToFileSetType(ctx context.Context, fileMap map[string]InstanceFileModel) (types.Set, diag.Diagnostics) {
	files := make([]InstanceFileModel, 0, len(fileMap))
	for _, v := range fileMap {
		files = append(files, v)
	}

	return types.SetValueFrom(ctx, types.ObjectType{}, files)
}

// InstanceFileDelete deletes a file from an instance.
func InstanceFileDelete(server incus.InstanceServer, instanceName string, targetPath string) error {
	targetPath, err := toAbsFilePath(targetPath)
	if err != nil {
		return err
	}

	err = server.DeleteInstanceFile(instanceName, targetPath)
	if err != nil && !errors.IsNotFoundError(err) {
		return err
	}

	return nil
}

// InstanceFileUpload uploads a file to an instance.
func InstanceFileUpload(server incus.InstanceServer, instanceName string, file InstanceFileModel) error {
	content := file.Content.ValueString()
	sourcePath := file.SourcePath.ValueString()

	if content != "" && sourcePath != "" {
		return fmt.Errorf("File %q and %q are mutually exclusive.", "content", "source_path")
	}

	targetPath, err := toAbsFilePath(file.TargetPath.ValueString())
	if err != nil {
		return err
	}

	fileMode := file.Mode.ValueString()
	if fileMode == "" {
		fileMode = "0755"
	}

	mode, err := strconv.ParseUint(fileMode, 8, 32)
	if err != nil {
		return fmt.Errorf("Failed to parse file mode: %v", err)
	}

	// Build the file creation request, without the content.
	args := &incus.InstanceFileArgs{
		Type: "file",
		Mode: int(mode),
		UID:  file.UserID.ValueInt64(),
		GID:  file.GroupID.ValueInt64(),
	}

	if file.Append.ValueBool() {
		args.WriteMode = "append"
	} else {
		args.WriteMode = "overwrite"
	}

	// If content was specified, read the string.
	if content != "" {
		args.Content = strings.NewReader(content)
	}

	// If a source was specified, read the contents of the source file.
	if sourcePath != "" {
		path, err := homedir.Expand(sourcePath)
		if err != nil {
			return fmt.Errorf("Unable to determine source file path: %v", err)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("Unable to read source file: %v", err)
		}
		defer f.Close()

		args.Content = f
	}

	if file.CreateDirs.ValueBool() {
		err := recursiveMkdir(server, instanceName, path.Dir(targetPath), *args)
		if err != nil {
			return fmt.Errorf("Could not create directories for file %q: %v", targetPath, err)
		}
	}

	err = server.CreateInstanceFile(instanceName, targetPath, *args)
	if err != nil {
		return fmt.Errorf("Could not upload file %q: %v", targetPath, err)
	}

	return nil
}

// recursiveMkdir recursively creates directories on target instance.
// This was copied almost as-is from github.com/lxc/incus/blob/main/lxc/file.go.
func recursiveMkdir(server incus.InstanceServer, instanceName string, p string, args incus.InstanceFileArgs) error {
	// Special case, every instance has a /, so there is nothing to do.
	if p == "/" {
		return nil
	}

	// Remove trailing "/" e.g. /A/B/C/. Otherwise we will end up with an
	// empty array entry "" which will confuse the Mkdir() loop below.
	pclean := filepath.Clean(p)
	parts := strings.Split(pclean, "/")
	i := len(parts)

	for ; i >= 1; i-- {
		cur := filepath.Join(parts[:i]...)
		_, resp, err := server.GetInstanceFile(instanceName, cur)
		if err != nil {
			continue
		}

		if resp.Type != "directory" {
			return fmt.Errorf("%s is not a directory", cur)
		}

		i++
		break
	}

	// Use same arguments as for file upload, only change file type.
	dirArgs := incus.InstanceFileArgs{
		Type: "directory",
		Mode: args.Mode,
		UID:  args.UID,
		GID:  args.GID,
	}

	for ; i <= len(parts); i++ {
		cur := filepath.Join(parts[:i]...)
		if cur == "" {
			continue
		}

		err := server.CreateInstanceFile(instanceName, cur, dirArgs)
		if err != nil {
			return err
		}
	}

	return nil
}

// toAbsFilePath returns absolute path of the given path and ensures that
// the path is not a directory.
func toAbsFilePath(path string) (string, error) {
	targetPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("Failed to determine absoulute target file path: %v", err)
	}

	isDir := strings.HasSuffix(targetPath, "/")
	if isDir {
		return "", fmt.Errorf("Target file %q cannot be a directory: %v", targetPath, err)
	}

	return targetPath, nil
}

package common

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type importMetadataTest struct {
	ImportID     string
	ResourceName string
	Fields       []string
	Options      []string
	Result       map[string]string
	ErrorString  string
}

func runTest(t *testing.T, test importMetadataTest) {
	t.Run(fmt.Sprintf("ImportID:%q", test.ImportID), func(t *testing.T) {
		meta := ImportMetadata{
			ResourceName:   test.ResourceName,
			RequiredFields: test.Fields,
			AllowedOptions: test.Options,
		}

		result, diag := meta.ParseImportID(test.ImportID)

		err := "<nil>"
		if diag != nil {
			// Error is the first line of the diagnostic detail.
			err = strings.SplitN(diag.Detail(), "\n", 2)[0]
		}

		// If ErrorString is not empty expect error.
		if test.ErrorString != "" {
			assert.Equal(t, test.ErrorString, err)
			return
		}

		if diag != nil {
			t.Error(err)
		}

		assert.Equal(t, test.Result, result)
	})
}

func TestSplitImportID_basic(t *testing.T) {
	tests := []importMetadataTest{
		{
			ImportID: "vm",
			Result:   map[string]string{"name": "vm"},
		},
		{
			ImportID: "/vm",
			Result:   map[string]string{"name": "vm"},
		},
		{
			ImportID: ":vm",
			Result:   map[string]string{"name": "vm"},
		},
		{
			ImportID: ":/vm",
			Result:   map[string]string{"name": "vm"},
		},
		{
			ImportID: "proj/vm",
			Result:   map[string]string{"name": "vm", "project": "proj"},
		},
		{
			ImportID: ":proj/vm",
			Result:   map[string]string{"name": "vm", "project": "proj"},
		},
		{
			ImportID: "rem:vm",
			Result:   map[string]string{"name": "vm", "remote": "rem"},
		},
		{
			ImportID: "rem:/vm",
			Result:   map[string]string{"name": "vm", "remote": "rem"},
		},
		{
			ImportID: "rem:proj/vm",
			Result:   map[string]string{"name": "vm", "project": "proj", "remote": "rem"},
		},
		{
			ImportID:    "",
			ErrorString: "Import ID cannot be empty.",
		},
	}

	for _, test := range tests {
		test.Fields = []string{"name"}
		runTest(t, test)
	}
}

func TestSplitImportID_RequiredFields(t *testing.T) {
	tests := []importMetadataTest{
		{
			ImportID: "vm",
			Fields:   []string{"name"},
			Result: map[string]string{
				"name": "vm",
			},
		},
		{
			ImportID: ":/vm",
			Fields:   []string{"name"},
			Result: map[string]string{
				"name": "vm",
			},
		},
		{
			ImportID: "project/vm",
			Fields:   []string{"name"},
			Result: map[string]string{
				"name":    "vm",
				"project": "project",
			},
		},

		{
			ImportID: "/vm/test",
			Fields:   []string{"name", "surname"},
			Result: map[string]string{
				"name":    "vm",
				"surname": "test",
			},
		},
		{
			ImportID: "remote:/vm/test",
			Fields:   []string{"name", "surname"},
			Result: map[string]string{
				"name":    "vm",
				"surname": "test",
				"remote":  "remote",
			},
		},
		{
			ImportID: "remote:project/vm/test",
			Fields:   []string{"name", "surname"},
			Result: map[string]string{
				"name":    "vm",
				"surname": "test",
				"project": "project",
				"remote":  "remote",
			},
		},
		{
			ImportID:    ":",
			Fields:      []string{"name"},
			ErrorString: "Import ID requires non-empty value for \"name\".",
		},
		{
			ImportID:    "vm",
			Fields:      []string{"name", "surname"},
			ErrorString: "Import ID does not contain all required fields: [name, surname].",
		},
	}

	for _, test := range tests {
		runTest(t, test)
	}
}

func TestSplitImportID_AllowedOption(t *testing.T) {
	tests := []importMetadataTest{
		{
			ImportID: "vm,image=asd",
			Fields:   []string{"name"},
			Options:  []string{"image"},
			Result: map[string]string{
				"name":  "vm",
				"image": "asd",
			},
		},
		{
			ImportID: "vm,,,image=asd,,,,",
			Fields:   []string{"name"},
			Options:  []string{"image"},
			Result: map[string]string{
				"name":  "vm",
				"image": "asd",
			},
		},
		{
			ImportID: "vm,image=",
			Fields:   []string{"name"},
			Options:  []string{"image"},
			Result: map[string]string{
				"name":  "vm",
				"image": "",
			},
		},
		{
			ImportID: "remote:project/vm",
			Fields:   []string{"name"},
			Options:  []string{"image", "size"},
			Result: map[string]string{
				"remote":  "remote",
				"project": "project",
				"name":    "vm",
			},
		},
		{
			ImportID: "remote:project/vm,image=jammy,size=5GiB",
			Fields:   []string{"name"},
			Options:  []string{"image", "size"},
			Result: map[string]string{
				"remote":  "remote",
				"project": "project",
				"name":    "vm",
				"image":   "jammy",
				"size":    "5GiB",
			},
		},
		{
			ImportID: "remote:/vm/123,image=jammy",
			Fields:   []string{"name", "surname"},
			Options:  []string{"image", "size"},
			Result: map[string]string{
				"remote":  "remote",
				"name":    "vm",
				"surname": "123",
				"image":   "jammy",
			},
		},
		{
			ImportID:    "vm,image",
			Fields:      []string{"name"},
			Options:     []string{"image", "size"},
			ErrorString: "Import ID contains invalid option \"image\". Options must be in key=value format.",
		},
		{
			ImportID:    "vm,image=asd",
			Fields:      []string{"name"},
			Options:     []string{"size"},
			ErrorString: "Import ID contains unexpected option \"image\".",
		},
	}

	for _, test := range tests {
		runTest(t, test)
	}
}

func TestSplitImportID_ErrorFormat(t *testing.T) {
	tests := []importMetadataTest{
		{
			ImportID:    "",
			Fields:      []string{"name"},
			ErrorString: "[<remote>:][<project>/]<name>",
		},
		{
			ImportID:    "",
			Fields:      []string{"name", "surname"},
			ErrorString: "[<remote>:][<project>]/<name>/<surname>",
		},
		{
			ImportID:    "",
			Fields:      []string{"name"},
			Options:     []string{"image"},
			ErrorString: "[<remote>:][<project>/]<name>[,image=<value>]",
		},
		{
			ImportID:    "",
			Fields:      []string{"name", "surname"},
			Options:     []string{"image"},
			ErrorString: "[<remote>:][<project>]/<name>/<surname>[,image=<value>]",
		},
		{
			ImportID:    "",
			Fields:      []string{"name", "surname"},
			Options:     []string{"image", "type"},
			ErrorString: "[<remote>:][<project>]/<name>/<surname>[,image=<value>][,type=<value>]",
		},
		{
			ImportID:    "",
			Fields:      []string{"name"},
			Options:     []string{"image", "type"},
			ErrorString: "[<remote>:][<project>/]<name>[,image=<value>][,type=<value>]",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("ImportID:%q", test.ImportID), func(t *testing.T) {
			meta := ImportMetadata{
				ResourceName:   test.ResourceName,
				RequiredFields: test.Fields,
				AllowedOptions: test.Options,
			}

			_, diag := meta.ParseImportID(test.ImportID)
			if diag == nil {
				t.Errorf("Expected an error, but received <nil>.")
				return
			}

			// Format is after lxd_<resName>.<resource> part.
			parts := strings.SplitN(diag.Detail(), "<resource> ", 2)
			if len(parts) != 2 {
				t.Errorf("Unexpected error: %q", diag.Detail())
				return
			}

			format := parts[1]
			assert.Equal(t, test.ErrorString, format)
		})
	}
}

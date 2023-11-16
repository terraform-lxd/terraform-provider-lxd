package acctest

import (
	"testing"
)

// PreCheck is a precheck that ensures test requirements, such as existing
// environment variables, are met. It should be included in every acc test.
func PreCheck(t *testing.T) {
	// if os.Getenv("TEST_LXD_IS_VERSION") == "" {
	// 	t.Fatal("TEST_LXD_IS_VERSION must be set for acceptance tests")
	// }

	// if os.Getenv("TEST_LXD_HAS_VIRTUALIZATION") == "" {
	// 	t.Fatal("TEST_LXD_HAS_VIRTUALIZATION must be set for acceptance tests")
	// }

	// if os.Getenv("TEST_LXD_IS_CLUSTERED") == "" {
	// 	t.Fatal("TEST_LXD_IS_CLUSTERED must be set for acceptance tests")
	// }

	// if os.Getenv("TEST_LXD_EXTENSIONS") == "" {
	// 	t.Fatal("TEST_LXD_EXTENSIONS must be set for acceptance tests")
	// }
}

// PreCheckLxdVersion skips the test if the server's version does not satisfy
// the provided version constraints. The version constraints are detailed at:
// https://pkg.go.dev/github.com/hashicorp/go-version#readme-version-constraints
// func PreCheckLxdVersion(t *testing.T, versionConstraint string) {
// 	server, _, err := GetTestInstanceServer(t).GetServer()
// 	if err != nil {
// 		t.Fatalf("Failed to retrieve the server: %v", err)
// 	}

// 	serverVersion := server.Environment.ServerVersion
// 	ok, err := utils.CheckVersion(serverVersion, versionConstraint)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if !ok {
// 		t.Skipf("Test %q skipped. LXD version %q does not satisfy the version constraint %q", t.Name(), serverVersion, versionConstraint)
// 	}
// }

// PreCheckAPIExtensions skips the test if the LXD server does not support
// the required extensions.
// func PreCheckAPIExtensions(t *testing.T, extensions []string) {
// 	instServer := GetTestInstanceServer(t)

// 	missing := []string{}
// 	for _, e := range extensions {
// 		if !instServer.HasExtension(e) {
// 			missing = append(missing, e)
// 		}
// 	}

// 	if len(missing) > 0 {
// 		t.Skipf("Test %q skipped. Missing required extensions: %v", t.Name(), missing)
// 	}
// }

// PreCheckVirtualization skips the test if the LXD server does not support
// virtualization.
// func PreCheckVirtualization(t *testing.T) {
// 	server, _, err := GetTestInstanceServer(t).GetServer()
// 	if err != nil {
// 		t.Fatalf("Failed to retrieve the server: %v", err)
// 	}

// 	// Ensure that LXD server supports qemu driver which is required
// 	// for virtualization.
// 	if !strings.Contains(server.Environment.Driver, "qemu") {
// 		t.Skipf("Test %q skipped. Server does not support virtualization.", t.Name())
// 	}
// }

// PreCheckClustering skips the test if LXD server is not running in
// clustered mode.
// func PreCheckClustering(t *testing.T) {
// 	instServer := GetTestInstanceServer(t)
// 	if !instServer.IsClustered() {
// 		t.Skipf("Test %q skipped. Server is not running in clustered mode.", t.Name())
// 	}
// }

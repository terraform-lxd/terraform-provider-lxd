package auth_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccIdentityToken_bearer(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "1H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "identity", identity),
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "expiry", "1H"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
			{
				// Change expiry to issue a new token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "2H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "expiry", "2H"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
		},
	})
}

func TestAccIdentityToken_devlxd(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "auth_bearer_devlxd", "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "devlxd", "30d"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "identity", identity),
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "expiry", "30d"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
		},
	})
}

func TestAccIdentityToken_devlxdSocket(t *testing.T) {
	identity := acctest.GenerateName(2, "-")
	group := acctest.GenerateName(2, "-")
	instance := acctest.GenerateName(2, "-")
	pool := acctest.GenerateName(2, "-")
	network := acctest.GenerateName(2, "-")
	volume := acctest.GenerateName(2, "-")

	instanceName := "lxd_instance.instance"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "auth_bearer_devlxd", "access_management_expiry", "devlxd_volume_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// The token of a devlxd identity must be accepted by the DevLXD
				// API, which is verified by listing the volumes of an empty
				// storage pool, creating one from within the instance, and
				// attaching it to that same instance. Both the volume and the
				// device are owned by the identity that created them.
				Config: acctest.Provider() + testAccIdentityToken_devlxdSocket(identity, group, instance, pool, network, volume, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttr(instanceName, "status", "Running"),
					resource.TestCheckResourceAttr(instanceName, "execs.01-install-curl.exit_code", "0"),
					resource.TestCheckResourceAttr(instanceName, "execs.02-list-volumes.exit_code", "0"),
					resource.TestCheckResourceAttr(instanceName, "execs.02-list-volumes.stdout", "200 []"),
					resource.TestCheckResourceAttr(instanceName, "execs.03-create-volume.exit_code", "0"),
					resource.TestMatchResourceAttr(instanceName, "execs.03-create-volume.stdout", regexp.MustCompile(`"name":"`+volume+`"`)),
					resource.TestMatchResourceAttr(instanceName, "execs.03-create-volume.stdout", regexp.MustCompile(`"volatile\.devlxd\.owner":`)),
					resource.TestCheckResourceAttr(instanceName, "execs.04-attach-volume.exit_code", "0"),
					resource.TestMatchResourceAttr(instanceName, "execs.04-attach-volume.stdout", regexp.MustCompile(`"source":"`+volume+`"`)),

					// The attached device is owned by the devlxd identity, so it
					// is not recorded in the instance state.
					resource.TestCheckResourceAttr(instanceName, "device.#", "1"),
					resource.TestCheckResourceAttr(instanceName, "device.0.name", "eth0"),
				),
			},
			{
				// Neither the volume nor the device that attaches it to the
				// instance is managed by Terraform, therefore they produce no
				// drift.
				Config: acctest.Provider() + testAccIdentityToken_devlxdSocket(identity, group, instance, pool, network, volume, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				// Terraform cannot remove a volume it does not manage, and the
				// storage pool cannot be destroyed while the volume exists, so
				// the volume is detached and removed over the DevLXD socket.
				Config: acctest.Provider() + testAccIdentityToken_devlxdSocket(identity, group, instance, pool, network, volume, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(instanceName, "execs.05-detach-volume.exit_code", "0"),
					resource.TestCheckResourceAttr(instanceName, "execs.06-delete-volume.exit_code", "0"),
				),
			},
		},
	})
}

func TestAccIdentityToken_revoked(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "1H"),
			},
			{
				// A token that is revoked outside of Terraform is no longer
				// reported by the server and must be reissued.
				PreConfig: func() {
					server := acctest.InstanceServer(t)

					err := server.RevokeBearerIdentityToken(identity)
					if err != nil {
						t.Fatalf("Failed to revoke token of identity %q: %v", identity, err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("lxd_auth_identity_token.token", plancheck.ResourceActionCreate),
					},
				},
			},
			{
				// Reissue token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "1H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
		},
	})
}

func TestAccIdentityToken_expired(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue a short lived token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "10S"),
			},
			{
				// An expired token must be reissued, even though the server
				// keeps reporting its expiry.
				PreConfig: func() {
					time.Sleep(11 * time.Second)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("lxd_auth_identity_token.token", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccIdentityToken_destroyed(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token and read the identity that bears it. The identity
				// must report the expiry of the issued token.
				Config: acctest.Provider() + testAccIdentityToken_identityDataSource(identity, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttrWith("data.lxd_auth_identity.identity", "expires_at", checkExpiresInFuture),
					resource.TestCheckResourceAttrPair("data.lxd_auth_identity.identity", "expires_at", "lxd_auth_identity_token.token", "expires_at"),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "pending", "false"),
				),
			},
			{
				// Remove just the token resource, which revokes the token while
				// the identity that bears it remains.
				Config: acctest.Provider() + testAccIdentityToken_identityDataSource(identity, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
				),
			},
			{
				// Read the identity once more, now that its token is revoked.
				// The identity must report no expiry.
				Config: acctest.Provider() + testAccIdentityToken_identityDataSource(identity, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
					resource.TestCheckNoResourceAttr("data.lxd_auth_identity.identity", "expires_at"),
				),
			},
		},
	})
}

func TestAccIdentityToken_serverExpiry(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token without an explicit expiry, which makes the
				// server apply its default.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "identity", identity),
					resource.TestCheckNoResourceAttr("lxd_auth_identity_token.token", "expiry"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
		},
	})
}

func TestAccIdentityToken_createBeforeDestroy(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token.
				Config: acctest.Provider() + testAccIdentityToken_createBeforeDestroy(identity, "1H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
			{
				// Ensure token created with create-before-destroy (or outside the Terraform)
				// is not revoked.
				Config: acctest.Provider() + testAccIdentityToken_createBeforeDestroy(identity, "2H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "expiry", "2H"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
		},
	})
}

// checkExpiresInFuture ensures the expiry is an RFC3339 timestamp in the future.
func checkExpiresInFuture(value string) error {
	expiresAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return fmt.Errorf("Expected an RFC3339 timestamp, got %q: %w", value, err)
	}

	if !expiresAt.After(time.Now()) {
		return fmt.Errorf("Expected expiry %q to be in the future", value)
	}

	return nil
}

// devlxdListVolumes queries the volumes of the given storage pool over the
// DevLXD socket using the token from the TOKEN environment variable. It prints
// the response status with the response body, and fails if the status is not
// 200. An authorization failure is therefore reported as a failed exec.
//
// The curl write-out format is escaped as "%%{" because HCL would otherwise
// interpret "%{" as a template directive.
const devlxdListVolumes = `
status=$(curl \
  -s \
  -o /tmp/volumes \
  -w '%%{http_code}' \
  --unix-socket /dev/lxd/sock \
  -H "Authorization: Bearer $TOKEN" \
  "http://localhost/1.0/storage-pools/$POOL/volumes?recursion=1")

printf '%s %s' "$status" "$(cat /tmp/volumes)"
test "$status" = "200"
`

// devlxdCreateVolume creates a custom storage volume over the DevLXD socket and
// prints the volumes that the identity owns in the pool. Volume creation is
// asynchronous, therefore the volumes are polled for until the new one appears.
const devlxdCreateVolume = `
curl \
  -s \
  -f \
  -o /dev/null \
  --unix-socket /dev/lxd/sock \
  -H "Authorization: Bearer $TOKEN" \
  -X POST \
  --data "{\"name\": \"$VOLUME\", \"type\": \"custom\", \"content_type\": \"filesystem\"}" \
  "http://localhost/1.0/storage-pools/$POOL/volumes"

for _ in $(seq 10); do
  volumes=$(curl -s --unix-socket /dev/lxd/sock -H "Authorization: Bearer $TOKEN" \
    "http://localhost/1.0/storage-pools/$POOL/volumes?recursion=1")

  echo "$volumes" | grep -q "\"name\":\"$VOLUME\"" && break
  sleep 1
done

printf '%s' "$volumes"
echo "$volumes" | grep -q "\"name\":\"$VOLUME\""
`

// devlxdAttachVolume attaches the storage volume to the instance it was created
// from over the DevLXD socket, and prints the instance devices that the identity
// owns. Device attachment is asynchronous, therefore the devices are polled for
// until the new one appears.
const devlxdAttachVolume = `
curl \
  -s \
  -f \
  -o /dev/null \
  --unix-socket /dev/lxd/sock \
  -H "Authorization: Bearer $TOKEN" \
  -X PATCH \
  --data "{\"devices\": {\"$VOLUME\": {\"type\": \"disk\", \"pool\": \"$POOL\", \"source\": \"$VOLUME\", \"path\": \"/mnt/$VOLUME\"}}}" \
  "http://localhost/1.0/instances/$INSTANCE"

for _ in $(seq 10); do
  instance=$(curl -s --unix-socket /dev/lxd/sock -H "Authorization: Bearer $TOKEN" \
    "http://localhost/1.0/instances/$INSTANCE")

  echo "$instance" | grep -q "\"source\":\"$VOLUME\"" && break
  sleep 1
done

printf '%s' "$instance"
echo "$instance" | grep -q "\"source\":\"$VOLUME\""
`

// devlxdDetachVolume detaches the storage volume from the instance over the
// DevLXD socket. Device detachment is asynchronous, therefore the devices are
// polled for until the removed one is gone.
const devlxdDetachVolume = `
curl \
  -s \
  -f \
  -o /dev/null \
  --unix-socket /dev/lxd/sock \
  -H "Authorization: Bearer $TOKEN" \
  -X PATCH \
  --data "{\"devices\": {\"$VOLUME\": null}}" \
  "http://localhost/1.0/instances/$INSTANCE"

for _ in $(seq 10); do
  instance=$(curl -s --unix-socket /dev/lxd/sock -H "Authorization: Bearer $TOKEN" \
    "http://localhost/1.0/instances/$INSTANCE")

  echo "$instance" | grep -q "\"source\":\"$VOLUME\"" || exit 0
  sleep 1
done

exit 1
`

// devlxdDeleteVolume removes the storage volume over the DevLXD socket. Volume
// deletion is asynchronous, therefore the volumes are polled for until the
// removed one is gone.
const devlxdDeleteVolume = `
curl \
  -s \
  -f \
  -o /dev/null \
  --unix-socket /dev/lxd/sock \
  -H "Authorization: Bearer $TOKEN" \
  -X DELETE \
  "http://localhost/1.0/storage-pools/$POOL/volumes/custom/$VOLUME"

for _ in $(seq 10); do
  volumes=$(curl -s --unix-socket /dev/lxd/sock -H "Authorization: Bearer $TOKEN" \
    "http://localhost/1.0/storage-pools/$POOL/volumes?recursion=1")

  echo "$volumes" | grep -q "\"name\":\"$VOLUME\"" || exit 0
  sleep 1
done

exit 1
`

func testAccIdentityToken_devlxdSocket(name string, group string, instance string, pool string, network string, volume string, removeVolume bool) string {
	removeExecs := ""
	if removeVolume {
		removeExecs = fmt.Sprintf(`
		    "05-detach-volume" = {
		      command       = ["sh", "-c", %q]
		      trigger       = "once"
		      fail_on_error = true
		      record_output = true

		      environment = {
		        TOKEN    = lxd_auth_identity_token.token.token
		        INSTANCE = %q
		        VOLUME   = %q
		      }
		    }

		    "06-delete-volume" = {
		      command       = ["sh", "-c", %q]
		      trigger       = "once"
		      fail_on_error = true
		      record_output = true

		      environment = {
		        TOKEN  = lxd_auth_identity_token.token.token
		        POOL   = lxd_storage_pool.pool.name
		        VOLUME = %q
		      }
		    }
		`, devlxdDetachVolume, instance, volume, devlxdDeleteVolume, volume)
	}

	return fmt.Sprintf(`
		resource "lxd_storage_pool" "pool" {
		  name   = %q
		  driver = "dir"
		}

		# The instance needs egress to install a client that can talk to the
		# DevLXD socket, therefore a network is created for it. The IPv4 subnet
		# is left to the server, which assigns a free one with NAT enabled.
		resource "lxd_network" "network" {
		  name = %q

		  config = {
		    "ipv6.address" = "none"
		  }
		}

		resource "lxd_auth_group" "group" {
		  name = %q

		  # Volume management and instance editing are granted at the project
		  # level, because the storage pool entity type provides no volume
		  # related entitlements, and the instance does not exist yet when the
		  # group is created.
		  permissions = [
		    {
		      entitlement = "can_view"
		      entity_type = "project"
		      entity_args = {
		        name = "default"
		      }
		    },
		    {
		      entitlement = "storage_volume_manager"
		      entity_type = "project"
		      entity_args = {
		        name = "default"
		      }
		    },
		    {
		      entitlement = "can_edit_instances"
		      entity_type = "project"
		      entity_args = {
		        name = "default"
		      }
		    },
		  ]
		}

		resource "lxd_auth_identity" "identity" {
		  type   = "devlxd"
		  name   = %q
		  groups = [lxd_auth_group.group.name]
		}

		resource "lxd_auth_identity_token" "token" {
		  identity = lxd_auth_identity.identity.name
		  expiry   = "1H"
		}

		resource "lxd_instance" "instance" {
		  name  = %q
		  image = %q

		  config = {
		    "security.devlxd.management.volumes" = "true"
		  }

		  device {
		    name = "eth0"
		    type = "nic"
		    properties = {
		      nictype = "bridged"
		      parent  = lxd_network.network.name
		    }
		  }

		  # Execs must not run before the instance can reach the network.
		  wait_for {
		    type = "ipv4"
		  }

		  execs = {
		    "01-install-curl" = {
		      command       = ["apk", "add", "--no-cache", "curl"]
		      trigger       = "once"
		      fail_on_error = true
		    }

		    "02-list-volumes" = {
		      command       = ["sh", "-c", %q]
		      trigger       = "once"
		      fail_on_error = true
		      record_output = true

		      environment = {
		        TOKEN = lxd_auth_identity_token.token.token
		        POOL  = lxd_storage_pool.pool.name
		      }
		    }

		    "03-create-volume" = {
		      command       = ["sh", "-c", %q]
		      trigger       = "once"
		      fail_on_error = true
		      record_output = true

		      environment = {
		        TOKEN  = lxd_auth_identity_token.token.token
		        POOL   = lxd_storage_pool.pool.name
		        VOLUME = %q
		      }
		    }

		    "04-attach-volume" = {
		      command       = ["sh", "-c", %q]
		      trigger       = "once"
		      fail_on_error = true
		      record_output = true

		      environment = {
		        TOKEN    = lxd_auth_identity_token.token.token
		        INSTANCE = %q
		        POOL     = lxd_storage_pool.pool.name
		        VOLUME   = %q
		      }
		    }
		    %s
		  }
		}
	`,
		pool,
		network,
		group,
		name,
		instance,
		acctest.TestImage,
		devlxdListVolumes,
		devlxdCreateVolume,
		volume,
		devlxdAttachVolume,
		instance,
		volume,
		removeExecs,
	)
}

func testAccIdentityToken_identityDataSource(name string, withToken bool) string {
	token := ""
	dependsOn := ""

	if withToken {
		token = `
		resource "lxd_auth_identity_token" "token" {
		  identity = lxd_auth_identity.identity.name
		  expiry   = "1H"
		}
		`

		// Without an explicit dependency the identity is read before the
		// token is issued, and therefore reports no expiry.
		dependsOn = "depends_on = [lxd_auth_identity_token.token]"
	}

	return fmt.Sprintf(`
		resource "lxd_auth_identity" "identity" {
		  type = "bearer"
		  name = %q
		}
		%s
		data "lxd_auth_identity" "identity" {
		  type = "bearer"
		  name = lxd_auth_identity.identity.name

		  %s
		}
	`,
		name,
		token,
		dependsOn,
	)
}

func testAccIdentityToken_createBeforeDestroy(name string, expiry string) string {
	return fmt.Sprintf(`
		resource "lxd_auth_identity" "identity" {
		  type = "bearer"
		  name = %q
		}

		resource "lxd_auth_identity_token" "token" {
		  identity = lxd_auth_identity.identity.name
		  expiry   = %q

		  lifecycle {
		    create_before_destroy = true
		  }
		}
	`,
		name,
		expiry,
	)
}

func testAccIdentityToken(name string, identityType string, expiry string) string {
	tokenExpiry := ""
	if expiry != "" {
		tokenExpiry = fmt.Sprintf("expiry = %q", expiry)
	}

	return fmt.Sprintf(`
		resource "lxd_auth_identity" "identity" {
		  type = %q
		  name = %q
		}

		resource "lxd_auth_identity_token" "token" {
		  identity = lxd_auth_identity.identity.name
		  %s
		}
	`,
		identityType,
		name,
		tokenExpiry,
	)
}

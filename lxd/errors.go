package lxd

import (
	"errors"
	"net/http"

	"github.com/canonical/lxd/shared/api"
)

var errNetworksNotImplemented = errors.New("This LXD server does not support " +
	"the creation of networks. You must be running LXD 2.3 or later for this " +
	"feature.")

func isNotFoundError(err error) bool {
	// For LXD versions below "4.0".
	if err.Error() == "No such object" || err.Error() == "not found" {
		return true
	}

	return api.StatusErrorCheck(err, http.StatusNotFound)
}

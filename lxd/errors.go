package lxd

import (
	"errors"
	"net/http"

	"github.com/canonical/lxd/shared/api"
)

func isNotFoundError(err error) bool {
	// For LXD versions below "4.0".
	if err.Error() == "No such object" || err.Error() == "not found" {
		return true
	}

	return api.StatusErrorCheck(err, http.StatusNotFound)
}

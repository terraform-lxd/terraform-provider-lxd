package lxd

import (
	"net/http"

	"github.com/canonical/lxd/shared/api"
)

func isNotFoundError(err error) bool {
	return api.StatusErrorCheck(err, http.StatusNotFound)
}

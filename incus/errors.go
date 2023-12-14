package incus

import (
	"net/http"

	"github.com/lxc/incus/shared/api"
)

func isNotFoundError(err error) bool {
	return api.StatusErrorCheck(err, http.StatusNotFound)
}

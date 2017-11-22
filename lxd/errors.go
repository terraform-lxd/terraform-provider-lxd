package lxd

import (
	"errors"
)

var errNetworksNotImplemented = errors.New("This LXD server does not support " +
	"the creation of networks. You must be running LXD 2.3 or later for this " +
	"feature.")

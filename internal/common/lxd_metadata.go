package common

import (
	"fmt"
	"sync"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"golang.org/x/sync/singleflight"
)

var (
	metadataCache     = make(map[string]*api.MetadataConfiguration)
	metadataCacheLock sync.RWMutex
	metadataGroup     singleflight.Group
)

func ServerMetadataConfiguration(name string, server lxd.InstanceServer) (*api.MetadataConfiguration, error) {
	metadataCacheLock.RLock()
	meta, ok := metadataCache[name]
	metadataCacheLock.RUnlock()
	if ok {
		return meta, nil
	}

	value, err, _ := metadataGroup.Do(name, func() (any, error) {
		metadataCacheLock.RLock()
		meta, ok := metadataCache[name]
		metadataCacheLock.RUnlock()
		if ok {
			return meta, nil
		}

		meta, err := server.GetMetadataConfiguration()
		if err != nil {
			return nil, err
		}

		metadataCacheLock.Lock()
		metadataCache[name] = meta
		metadataCacheLock.Unlock()

		return meta, nil
	})
	if err != nil {
		return nil, err
	}

	meta, ok = value.(*api.MetadataConfiguration)
	if !ok {
		return nil, fmt.Errorf("Unexpected metadata type %T", value)
	}

	return meta, nil
}

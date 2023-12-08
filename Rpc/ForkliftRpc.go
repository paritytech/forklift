package Rpc

import (
	"forklift/FileManager/Models"
	log "github.com/sirupsen/logrus"
	"sync"
)

type ForkliftRpc struct {
	Extern  map[string]bool
	Uploads chan Models.CacheItem
	lock    sync.RWMutex
}

func NewForkliftRpc() *ForkliftRpc {
	var uploads = make(chan Models.CacheItem, 20)
	var srv = ForkliftRpc{
		Extern:  make(map[string]bool),
		Uploads: uploads,
		lock:    sync.RWMutex{},
	}
	return &srv
}

func (server *ForkliftRpc) CheckExternDeps(paths *[]string, result *string) error {
	server.lock.RLock()
	defer server.lock.RUnlock()

	for _, path := range *paths {
		var _, b = server.Extern[path]
		if b {
			*result = path
			return nil
		}
	}
	*result = ""
	return nil
}

func (server *ForkliftRpc) RegisterExternDeps(paths *[]string, result *bool) error {
	server.lock.Lock()
	defer server.lock.Unlock()

	for _, path := range *paths {
		server.Extern[path] = true
	}
	*result = true
	return nil
}

func (server *ForkliftRpc) AddUpload(cacheItem Models.CacheItem, result *bool) error {
	server.Uploads <- cacheItem
	*result = true

	log.Debugf("Items in upload queue: %d", len(server.Uploads))
	return nil
}

package Rpc

import (
	"forklift/FileManager/Models"
	"sync"
)

type ForkliftRpc struct {
	Extern  map[string]bool
	Uploads chan Models.CacheItem
	lock    sync.RWMutex
}

func NewForkliftRpc() *ForkliftRpc {
	var srv = ForkliftRpc{
		Extern:  make(map[string]bool),
		Uploads: make(chan Models.CacheItem, 10),
		lock:    sync.RWMutex{},
	}
	return &srv
}

func (server *ForkliftRpc) CheckExternDeps(paths *[]string, result *bool) error {
	server.lock.RLock()
	defer server.lock.RUnlock()

	for _, path := range *paths {
		var _, b = server.Extern[path]
		if b {
			*result = true
			return nil
		}
	}
	*result = false
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
	return nil
}

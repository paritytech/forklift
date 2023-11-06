package Rpc

import (
	"sync"
)

var StopEvent = 1

type ForkliftRpc struct {
	Extern map[string]bool
	lock   sync.RWMutex
}

func NewForkliftRpc() *ForkliftRpc {
	var srv = ForkliftRpc{
		Extern: make(map[string]bool),
		lock:   sync.RWMutex{},
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

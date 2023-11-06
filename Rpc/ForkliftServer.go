package Rpc

import "sync"

type ForkliftRpcServer struct {
	Extern map[string]bool
	lock   sync.RWMutex
}

func NewForkliftServer() *ForkliftRpcServer {
	var srv = ForkliftRpcServer{
		Extern: make(map[string]bool),
		lock:   sync.RWMutex{},
	}
	return &srv
}

func (server *ForkliftRpcServer) CheckExternDeps(paths *[]string, result *bool) error {
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

func (server *ForkliftRpcServer) RegisterExternDeps(paths *[]string, result *bool) error {
	server.lock.Lock()
	defer server.lock.Unlock()

	for _, path := range *paths {
		server.Extern[path] = true
	}
	*result = true
	return nil
}

package Rpc

import (
	log "github.com/sirupsen/logrus"
	"net/rpc"
)

type ForkliftRpcClient struct {
	rpcClient *rpc.Client
}

func NewForkliftRpcClient() *ForkliftRpcClient {
	var forkliftClient = &ForkliftRpcClient{}
	var rpcClient, _ = rpc.Dial("tcp", ":9999")

	forkliftClient.rpcClient = rpcClient

	return forkliftClient
}

func (client *ForkliftRpcClient) RegisterExternDeps(deps *[]string) {
	if deps == nil {
		return
	}

	if len(*deps) == 0 {
		return
	}

	_ = client.rpcClient.Call("ForkliftRpcServer.RegisterExternDeps", deps, nil)
}

func (client *ForkliftRpcClient) CheckExternDeps(deps *[]string) bool {
	if deps == nil {
		return false
	}

	if len(*deps) == 0 {
		return false
	}

	log.Debug("$$$$$$$$$$$$$$$", *deps)
	var result bool
	err := client.rpcClient.Call("ForkliftRpcServer.CheckExternDeps", deps, &result)
	if err != nil {
		log.Fatal(err, *deps)
	}
	return result
}

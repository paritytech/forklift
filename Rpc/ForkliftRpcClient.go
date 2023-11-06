package Rpc

import (
	log "github.com/sirupsen/logrus"
	"net/rpc"
	"os"
	"path/filepath"
)

type ForkliftRpcClient struct {
	rpcClient *rpc.Client
}

func NewForkliftRpcClient() *ForkliftRpcClient {
	var forkliftClient = &ForkliftRpcClient{}
	var address string

	wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR")

	if !ok || wd == "" {
		address = "forklift.sock"
	} else {
		address = filepath.Join(wd, "forklift.sock")
	}

	var rpcClient, _ = rpc.Dial("unix", address)

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

	_ = client.rpcClient.Call("ForkliftRpc.RegisterExternDeps", deps, nil)
}

func (client *ForkliftRpcClient) CheckExternDeps(deps *[]string) bool {
	if deps == nil {
		return false
	}

	if len(*deps) == 0 {
		return false
	}

	var result bool
	err := client.rpcClient.Call("ForkliftRpc.CheckExternDeps", deps, &result)
	if err != nil {
		log.Fatal(err, *deps)
	}
	return result
}

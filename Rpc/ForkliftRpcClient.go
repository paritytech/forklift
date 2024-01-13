package Rpc

import (
	"forklift/FileManager/Models"
	RpcModels "forklift/Rpc/Models"
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

	var _, e = os.Stat(address)
	if e != nil {
		log.Fatal("No socket at "+address, e)
	}

	var rpcClient, err = rpc.Dial("unix", address)
	if err != nil {
		log.Fatal(err)
	}

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

func (client *ForkliftRpcClient) CheckExternDeps(deps *[]string) string {
	if deps == nil {
		return ""
	}

	if len(*deps) == 0 {
		return ""
	}

	var result string
	err := client.rpcClient.Call("ForkliftRpc.CheckExternDeps", deps, &result)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

// AddUpload - send cacheItem to server for upload
func (client *ForkliftRpcClient) AddUpload(cacheItem Models.CacheItem) {
	var result bool
	err := client.rpcClient.Call("ForkliftRpc.AddUpload", cacheItem, &result)
	if err != nil {
		log.Fatal(err)
	}
}

func (client *ForkliftRpcClient) ReportStatus(crateName string, status RpcModels.CrateCacheStatus) {
	var result bool

	var cacheStatusReport = RpcModels.CrateCacheStatusReport{
		CrateName:   crateName,
		CacheStatus: status,
	}

	err := client.rpcClient.Call("ForkliftRpc.ReportStatus", &cacheStatusReport, &result)
	if err != nil {
		log.Fatal(err)
	}
}

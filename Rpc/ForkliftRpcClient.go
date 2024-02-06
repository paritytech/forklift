package Rpc

import (
	"forklift/FileManager/Models"
	CacheUsage "forklift/Rpc/Models/CacheUsage"
	log "github.com/sirupsen/logrus"
	"net/rpc"
	"os"
)

type ForkliftRpcClient struct {
	rpcClient *rpc.Client
}

func NewForkliftRpcClient() *ForkliftRpcClient {
	var forkliftClient = &ForkliftRpcClient{}

	socketAddress, ok := os.LookupEnv("FORKLIFT_SOCKET")

	if !ok || socketAddress == "" {
		log.Warnf("FORKLIFT_SOCKET is not set, trying to use default socket 'forklift.sock'")
		socketAddress = "forklift.sock"
	}

	var _, e = os.Stat(socketAddress)
	if e != nil {
		log.Fatal("No socket at "+socketAddress, e)
	}

	var rpcClient, err = rpc.Dial("unix", socketAddress)
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

func (client *ForkliftRpcClient) ReportStatus(crateName string, status CacheUsage.Status) {
	var result bool

	var cacheStatusReport = CacheUsage.StatusReport{
		CrateName: crateName,
		Status:    status,
	}

	err := client.rpcClient.Call("ForkliftRpc.ReportStatus", &cacheStatusReport, &result)
	if err != nil {
		log.Error(err)
	}
}

func (client *ForkliftRpcClient) ReportStatusObject(report CacheUsage.StatusReport) {
	var result bool

	err := client.rpcClient.Call("ForkliftRpc.ReportStatus", &report, &result)
	if err != nil {
		log.Error(err)
	}
}

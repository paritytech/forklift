package Rpc

import (
	"errors"
	"forklift/FileManager/Models"
	log "forklift/Lib/Logging/ConsoleLogger"
	CacheUsage "forklift/Rpc/Models/CacheUsage"
	"net/rpc"
	"os"
)

type ForkliftRpcClient struct {
	rpcClient *rpc.Client
}

func NewForkliftRpcClient() *ForkliftRpcClient {
	var forkliftClient = &ForkliftRpcClient{}
	return forkliftClient
}

func (client *ForkliftRpcClient) Connect() error {
	socketAddress, ok := os.LookupEnv("FORKLIFT_SOCKET")

	if !ok || socketAddress == "" {
		log.Warningf("FORKLIFT_SOCKET is not set, trying to use default socket 'forklift.sock'")
		socketAddress = "forklift.sock"
	}

	var _, e = os.Stat(socketAddress)
	if e != nil {
		return errors.Join(errors.New("no socket at "+socketAddress), e)
	}

	var rpcClient, err = rpc.Dial("unix", socketAddress)
	if err != nil {
		return errors.Join(errors.New("unable to connect to socket"), err)
	}

	client.rpcClient = rpcClient

	return nil
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
		log.Fatalf(err.Error())
	}
	return result
}

// AddUpload - send cacheItem to server for upload
func (client *ForkliftRpcClient) AddUpload(cacheItem Models.CacheItem) {
	var result bool
	err := client.rpcClient.Call("ForkliftRpc.AddUpload", cacheItem, &result)
	if err != nil {
		log.Fatalf(err.Error())
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
		log.Errorf(err.Error())
	}
}

func (client *ForkliftRpcClient) ReportStatusObject(report CacheUsage.StatusReport) {
	var result bool

	err := client.rpcClient.Call("ForkliftRpc.ReportStatus", &report, &result)
	if err != nil {
		log.Errorf(err.Error())
	}
}

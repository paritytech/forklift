package Rpc

import (
	"errors"
	"net/rpc"
	"os"
	"path/filepath"
)

type ForkliftControlClient struct {
	rpcClient *rpc.Client
}

func NewForkliftControlClient() *ForkliftControlClient {
	var forkliftClient = &ForkliftControlClient{}
	return forkliftClient
}

func (client *ForkliftControlClient) Connect() error {
	var address string

	wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR")

	if !ok || wd == "" {
		address = "forklift.sock"
	} else {
		address = filepath.Join(wd, "forklift.sock")
	}

	var _, err = os.Stat(address)
	if err != nil {
		return errors.New("no forklift socket found")
	}

	rpcClient, err := rpc.Dial("unix", address)
	if err != nil {
		return errors.Join(errors.New("no forklift socket found"), err)
	}

	client.rpcClient = rpcClient
	return nil
}

/*
func (client *ForkliftControlClient) Stop() {
	_ = client.rpcClient.Call("ForkliftControlRpc.Stop", 0, nil)
}
*/

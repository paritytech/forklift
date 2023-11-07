package Rpc

import (
	log "github.com/sirupsen/logrus"
	"net"
	"net/rpc"
	"os"
)

type ForkliftRpcServer struct {
	goRpcServer *rpc.Server
}

func NewForkliftServer() *ForkliftRpcServer {
	var srv = ForkliftRpcServer{
		goRpcServer: rpc.NewServer(),
	}
	return &srv
}

func (server *ForkliftRpcServer) Start() {

	//check for existing server
	var _, e = os.Stat("forklift.sock")

	if e == nil {
		log.Error("Forklift RpcServer is already running for this location")
		os.Exit(0)
	}

	err := server.goRpcServer.Register(NewForkliftRpc())
	if err != nil {
		log.Fatalln(err)
	}

	socket, _ := net.Listen("unix", "forklift.sock")
	defer os.Remove("forklift.sock")

	server.goRpcServer.Accept(socket)
}

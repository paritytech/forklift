package Rpc

import (
	log "github.com/sirupsen/logrus"
	"net"
	"net/rpc"
	"os"
)

type ForkliftRpcServer struct {
	goRpcServer     *rpc.Server
	socket          net.Listener
	isStopRequested bool
	Channel         chan bool
}

func NewForkliftServer() *ForkliftRpcServer {
	var srv = ForkliftRpcServer{
		goRpcServer:     rpc.NewServer(),
		isStopRequested: false,
		Channel:         make(chan bool, 1),
	}
	return &srv
}

func (server *ForkliftRpcServer) Stop() {
	if isStopRequested {
		return
	}
	server.isStopRequested = true
	err := server.socket.Close()
	if err != nil {
		log.Error(err)
	}
}

func (server *ForkliftRpcServer) Start() {

	//check for existing server
	var _, e = os.Stat("forklift.sock")

	if e == nil {
		log.Error("Forklift RpcServer is already running for this location")
		server.Channel <- true
		return
	}

	err := server.goRpcServer.Register(NewForkliftRpc())
	if err != nil {
		log.Fatalln(err)
	}

	var controlRpc = NewControlRpc()
	err = server.goRpcServer.Register(controlRpc)
	if err != nil {
		log.Fatalln(err)
	}

	server.socket, err = net.Listen("unix", "forklift.sock")
	if err != nil {
		log.Fatalln(err)
	}

	/*go func() {
		time.Sleep(time.Second * 3)
		log.Infof("Stopping server")
		server.Stop()
	}()*/

	for !controlRpc.IsStopRequested() && !server.isStopRequested {
		var con, e = server.socket.Accept()
		if e == nil {
			go server.goRpcServer.ServeConn(con)
		}
	}

	server.Channel <- true
}

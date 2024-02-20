package Rpc

import (
	"fmt"
	log "forklift/Lib/Logging/ConsoleLogger"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
)

type ForkliftRpcServer struct {
	goRpcServer     *rpc.Server
	socket          net.Listener
	isStopRequested bool
	Channel         chan bool
	SocketAddress   string
}

func NewForkliftServer() *ForkliftRpcServer {
	var srv = ForkliftRpcServer{
		goRpcServer:     rpc.NewServer(),
		isStopRequested: false,
		Channel:         make(chan bool, 1),
		SocketAddress: filepath.Join(
			os.TempDir(),
			fmt.Sprintf("forklift-%d.sock", os.Getpid()),
		),
	}
	return &srv
}

// Stop - stop rpc server
func (server *ForkliftRpcServer) Stop() {
	server.isStopRequested = true
	err := server.socket.Close()
	if err != nil {
		log.Errorf("%s", err)
	}
}

// Start - start rpc server
func (server *ForkliftRpcServer) Start(forkliftRpc *ForkliftRpc) {

	log.Infof("Starting server on %s", server.SocketAddress)

	err := server.goRpcServer.Register(forkliftRpc)
	if err != nil {
		log.Errorf("Format of service ForkliftRpc is not correct. %s", err)
	}

	server.socket, err = net.Listen("unix", server.SocketAddress)
	if err != nil {
		log.Errorf("Listen error: %s", err)
	}

	for !server.isStopRequested {
		var con, e = server.socket.Accept()
		if e == nil {
			go server.goRpcServer.ServeConn(con)
		}
	}
}

package Server

import (
	"forklift/Rpc"
	log "github.com/sirupsen/logrus"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
)

func Run(args []string) {
	var server = rpc.NewServer()
	err := server.Register(Rpc.NewForkliftServer())
	if err != nil {
		log.Fatalln(err)
	}
	socket, _ := net.Listen("tcp", ":9999")
	//defer os.Remove("forklift.sock")

	go server.Accept(socket)

	// set RUSTC_WRAPPER env var
	var flExecPath, _ = os.Executable()
	flExecPath, _ = filepath.EvalSymlinks(flExecPath)
	os.Setenv("RUSTC_WRAPPER", flExecPath)

	// set FORKLIFT_WORK_DIR env var
	var wd, _ = os.Getwd()
	os.Setenv("FORKLIFT_WORK_DIR", wd)

	// execute cargo
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	_ = cmd.Run()
}

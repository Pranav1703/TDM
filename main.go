package main

import (
	"os"
	"os/signal"
	"shareIt/internal/server"
	"syscall"
	"time"
	// 	"shareIt/internal/tui"
)

func main() {	

	shutdownSig := make(chan os.Signal, 1)

	signal.Notify(shutdownSig,os.Interrupt,syscall.SIGINT,syscall.SIGTERM)

	go func(){
		time.Sleep(2 * time.Second)
		server.SendFile(server.TestFile2)
	}()
	// tui.InitTui()
	server.StartTcpServer(shutdownSig)
}
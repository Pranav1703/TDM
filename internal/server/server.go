package server

import (
	"log"
	"net"
	"shareIt/internal/client"
)

func StartTcpServer() {
	listener,err := net.Listen("tcp","127.0.0.1:8000")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for{
		conn , err := listener.Accept()
		if err != nil {
			log.Fatal("listener err:",err)
		}

		client := &client.Client{
			Conn: conn,
		}

		go client.HandleReq()
	}
}
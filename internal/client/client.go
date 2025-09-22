package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

type Client struct{
	Conn net.Conn
}

func (C *Client)HandleReq(){
	reader := bufio.NewReader(C.Conn)
	for{
		msg,err := reader.ReadString('\n')
		if err!= nil {
			C.Conn.Close()
			log.Println("error reading from " ,C.Conn.RemoteAddr(),err)
			return
		}

		fmt.Printf("message recieved: %s\n",msg)
		C.Conn.Write([]byte("message Received\n"))
	}
}
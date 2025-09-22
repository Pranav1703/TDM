package main

import (
	"fmt"
	"shareIt/internal/server"
)

func main() {
	fmt.Println("-----------------ShareIt CLI.-------------------")
	server.StartTcpServer()
}
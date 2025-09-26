package main

import (
	"fmt"
	"shareIt/internal/server"
	"shareIt/internal/tui"
)

func main() {
	fmt.Println("-----------------ShareIt CLI.-------------------")
	go server.StartTcpServer()
	tui.InitTui()
}
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"shareIt/internal/server"
	"shareIt/internal/tui"
	"syscall"
	"time"
		"flag"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {	

	port := flag.Int("port", 8000, "The port for the TCP file server.")
	flag.Parse()
	tcpPort := *port 

	f, err := tea.LogToFile("debug.log", "shareit")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	myIP, err := server.GetOutboundIP()
	if err != nil {
		log.Printf("Could not get local IP, discovery may be unreliable: %v", err)
		myIP = "127.0.0.1" // Fallback
	}
	myAddr := fmt.Sprintf("%s:%d", myIP, tcpPort)
	log.Printf("Starting ShareIt. My address is: %s", myAddr)



	// Create the TUI model first.
	model := tui.InitialModel()
	// Then create the program with the model.
	p := tea.NewProgram(model, tea.WithAltScreen())

	// THE FIX: Inject the program reference into the model using a method.
	// This avoids the deadlock by not sending a message before the program is running.
	model.SetProgram(p)

	// --- Start Backend Services in Goroutines ---

	// Start the discovery service to announce our presence.
	go server.AnnounceService(myAddr)

	// Start listening for peers and send updates to the TUI.
	go server.ListenForPeers(p,myAddr)

	// --- Start the TCP Server (with graceful shutdown) ---
	shutdownSig := make(chan os.Signal, 1)
	signal.Notify(shutdownSig, os.Interrupt, syscall.SIGTERM)
	go server.StartTcpServer(shutdownSig, tcpPort, p)

	// --- Run the TUI ---
	// This is a blocking call and will run until the user quits.
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running TUI: %v", err)
	}

	// TUI has quit, so we can signal the server to shut down.
	log.Println("TUI has quit. Sending shutdown signal to server.")
	shutdownSig <- syscall.SIGTERM
	// Give the server a moment to shut down before the program exits.
	time.Sleep(1 * time.Second)
	log.Println("Exiting.")
}
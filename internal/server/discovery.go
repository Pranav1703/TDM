package server

import (
	"fmt"
	"log"
	"net"
	"shareIt/internal/utils"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	// This is a multicast address and port.
	multicastAddr    = "239.0.0.1:9999"
	// This is the "magic" message prefix to identify our app's announcements.
	messagePrefix    = "SHAREIT_DISCOVERY"
	// How often to send out announcements.
	announceInterval = 2 * time.Second
	// How long to wait before considering a peer offline.
	peerTimeout = 5 * time.Second
)

// AnnounceService starts shouting on the network that our service is available.
// It takes the port of our main TCP file server as an argument.
func AnnounceService(tcpPort int) {
	myIP, err := GetOutboundIP()
	if err != nil {
		log.Printf("Could not get local IP, announcements will be limited: %v", err)
		myIP = "127.0.0.1" // Fallback
	}
	message := fmt.Sprintf("%s|%s:%d", messagePrefix, myIP, tcpPort)
	addr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		log.Fatalf("Error resolving multicast address: %v", err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("Error dialing multicast address: %v", err)
	}
	defer conn.Close()

	log.Printf("Starting to announce service on %s", multicastAddr)
	for {
		_, err := conn.Write([]byte(message))
		if err != nil {
			log.Printf("Error sending announcement: %v", err)
		}
		time.Sleep(announceInterval)
	}
}

// ListenForPeers runs a continuous loop to find peers and send updates to the TUI.
func ListenForPeers(p *tea.Program) {
	addr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		log.Fatalf("Error resolving UDP addr for listener: %v", err)
	}
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("Error listening to multicast UDP: %v", err)
	}
	defer conn.Close()

	// A map to keep track of peers and their last seen time.
	peers := make(map[string]time.Time)
	var mu sync.Mutex

	// Goroutine to periodically prune stale peers that have timed out.
	go func() {
		for {
			time.Sleep(peerTimeout)
			mu.Lock()
			var changed bool
			for peer, lastSeen := range peers {
				if time.Since(lastSeen) > peerTimeout {
					delete(peers, peer)
					changed = true // Mark that the list has changed
				}
			}
			// If any peers were removed, send an updated list to the TUI.
			if changed {
				var currentPeers []string
				for peer := range peers {
					currentPeers = append(currentPeers, peer)
				}
				p.Send(utils.PeersUpdatedMsg{Peers: currentPeers})
			}
			mu.Unlock()
		}
	}()

	// Main loop to listen for announcements.
	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}

		message := string(buffer[:n])
		if strings.HasPrefix(message, messagePrefix) {
			parts := strings.Split(message, "|")
			if len(parts) == 2 {
				peerAddr := parts[1]
				mu.Lock()
				// Check if it's a new peer.
				_, exists := peers[peerAddr]
				// Update the last seen time for the peer.
				peers[peerAddr] = time.Now()
				// If the peer is new, send an immediate update to the TUI.
				if !exists {
					var currentPeers []string
					for peer := range peers {
						currentPeers = append(currentPeers, peer)
					}
					p.Send(utils.PeersUpdatedMsg{Peers: currentPeers})
				}
				mu.Unlock()
			}
		}
	}
}

// getOutboundIP is a helper to get the preferred outbound IP address of this machine.
func GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}


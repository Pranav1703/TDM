package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"shareIt/internal/utils"
	"strings"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/net/ipv4"
)

const (
	multicastAddr    = "239.0.0.1:9999"
	messagePrefix    = "SHAREIT_DISCOVERY"
	announceInterval = 2 * time.Second
	peerTimeout      = 5 * time.Second
)

// AnnounceService remains the same as your version.
func AnnounceService(myAddr string) {
	message := fmt.Sprintf("%s|%s", messagePrefix, myAddr)
	addr, err := net.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		log.Fatalf("Error resolving multicast address: %v", err)
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		log.Fatalf("Error dialing multicast address: %v", err)
	}
	defer conn.Close()

	log.Printf("Starting to announce my address (%s) on %s", myAddr, multicastAddr)
	for {
		_, err := conn.Write([]byte(message))
		if err != nil {
			log.Printf("Error sending announcement: %v", err)
		}
		time.Sleep(announceInterval)
	}
}

// ListenForPeers is updated to allow multiple listeners on the same port.
func ListenForPeers(p *tea.Program, myAddr string) {
	addr, err := net.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		log.Fatalf("Error resolving UDP addr for listener: %v", err)
	}

	// THE FIX: Use ListenConfig to set socket options before binding.
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				// Set SO_REUSEADDR to allow multiple instances to bind to the same address.
				opErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}

	// Use the ListenConfig to create the packet listener.
	l, err := lc.ListenPacket(context.Background(), "udp4", fmt.Sprintf("0.0.0.0:%d", addr.Port))
	if err != nil {
		log.Fatalf("Error listening for packets: %v", err)
	}
	defer l.Close()

	packetConn := ipv4.NewPacketConn(l)

	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Error getting network interfaces: %v", err)
	}

	var joined bool
	for _, iface := range interfaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagMulticast) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}
		if err := packetConn.JoinGroup(&iface, addr); err == nil {
			log.Printf("Successfully joined multicast group on interface: %s", iface.Name)
			joined = true
			// We join on all suitable interfaces instead of just the first.
		}
	}

	if !joined {
		log.Printf("Warning: Could not join multicast group on any suitable interface. Discovery may not work.")
	}

	if err := packetConn.SetMulticastLoopback(true); err != nil {
		log.Printf("Warning: could not enable multicast loopback: %v", err)
	}

	log.Printf("Listening for peer announcements on %s", multicastAddr)
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
					changed = true
					log.Printf("Peer timed out and was removed: %s", peer)
				}
			}
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
		n, _, _, err := packetConn.ReadFrom(buffer)
		if err != nil {
			log.Printf("Error reading from packet conn: %v", err)
			continue
		}

		message := string(buffer[:n])
		log.Printf("Received multicast message: \"%s\"", message)

		if strings.HasPrefix(message, messagePrefix) {
			parts := strings.Split(message, "|")
			if len(parts) == 2 {
				peerAddr := parts[1]
				log.Printf("Discovered a potential peer: %s", peerAddr)

				if peerAddr == myAddr {
					log.Printf("Ignoring own announcement from %s", myAddr)
					continue
				}

				mu.Lock()
				_, exists := peers[peerAddr]
				peers[peerAddr] = time.Now()
				if !exists {
					log.Printf("New peer found: %s. Sending update to TUI.", peerAddr)
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

// GetOutboundIP remains the same.
func GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}


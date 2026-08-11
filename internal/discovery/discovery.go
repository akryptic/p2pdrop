package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Packet defines the JSON structure broadcasted over UDP.
type Packet struct {
	ID         string `json:"id"`
	DeviceName string `json:"device_name"`
	HTTPPort   uint16 `json:"http_port"`
}

// Peer represents an active discovered node on the local network.
type Peer struct {
	ID         string    `json:"id"`
	DeviceName string    `json:"device_name"`
	IP         string    `json:"ip"`
	HTTPPort   uint16    `json:"http_port"`
	LastSeen   time.Time `json:"last_seen"`
}

// Engine manages peer discovery via UDP broadcast pings.
type Engine struct {
	selfID     string
	deviceName string
	httpPort   uint16
	udpPort    uint16

	conn *net.UDPConn

	peerLock sync.RWMutex
	peers    map[string]Peer

	cancel context.CancelFunc
}

// NewEngine constructs a discovery Engine instance with an initialized peer map.
func NewEngine(selfID string, deviceName string, httpPort uint16, udpPort uint16) *Engine {
	return &Engine{
		selfID:     selfID,
		deviceName: deviceName,
		httpPort:   httpPort,
		udpPort:    udpPort,
		peers:      make(map[string]Peer),
	}
}

// Start binds the socket and spawns background loops for broadcast, listen, and cleanup.
func (e *Engine) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	// SO_REUSEPORT allows multiple instances on the same host to bind to the discovery port.
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var err error
			controlErr := c.Control(func(fd uintptr) {
				err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
				if err != nil {
					_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
				}
			})
			if controlErr != nil {
				return controlErr
			}
			return err
		},
	}

	bindAddr := fmt.Sprintf("0.0.0.0:%d", e.udpPort)
	packetConn, err := lc.ListenPacket(ctx, "udp4", bindAddr)
	if err != nil {
		return fmt.Errorf("failed to bind UDP socket on %s: %w", bindAddr, err)
	}

	udpConn, ok := packetConn.(*net.UDPConn)
	if !ok {
		return fmt.Errorf("expected *net.UDPConn, got %T", packetConn)
	}
	e.conn = udpConn

	go e.announceLoop(ctx)
	go e.listenLoop(ctx)
	go e.reaperLoop(ctx)

	return nil
}

// Stop signals background routines to shut down and closes the UDP socket.
func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	if e.conn != nil {
		e.conn.Close()
	}
}

// GetPeers returns a thread-safe snapshot of all currently active peers.
func (e *Engine) GetPeers() []Peer {
	e.peerLock.RLock()
	defer e.peerLock.RUnlock()

	peerList := make([]Peer, 0, len(e.peers))
	for _, peer := range e.peers {
		peerList = append(peerList, peer)
	}
	return peerList
}

// announceLoop periodically sends UDP broadcast pings containing device identity.
func (e *Engine) announceLoop(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	broadcastAddr := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: int(e.udpPort),
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			pkt := Packet{
				ID:         e.selfID,
				DeviceName: e.deviceName,
				HTTPPort:   e.httpPort,
			}

			data, err := json.Marshal(pkt)
			if err != nil {
				continue
			}

			_, _ = e.conn.WriteToUDP(data, broadcastAddr)
		}
	}
}

// listenLoop reads incoming UDP broadcast packets and updates active peer state.
func (e *Engine) listenLoop(ctx context.Context) {
	buf := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, srcAddr, err := e.conn.ReadFromUDP(buf)
			if err != nil {
				return
			}

			var pkt Packet
			if err := json.Unmarshal(buf[:n], &pkt); err != nil {
				continue
			}

			// Ignore self-broadcast pings
			if pkt.ID == e.selfID {
				continue
			}

			e.peerLock.Lock()
			e.peers[pkt.ID] = Peer{
				ID:         pkt.ID,
				DeviceName: pkt.DeviceName,
				IP:         srcAddr.IP.String(),
				HTTPPort:   pkt.HTTPPort,
				LastSeen:   time.Now(),
			}
			e.peerLock.Unlock()
		}
	}
}

// reaperLoop periodically removes peers that have stopped broadcasting past the timeout threshold.
func (e *Engine) reaperLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			e.peerLock.Lock()
			for id, peer := range e.peers {
				if time.Since(peer.LastSeen) > 10*time.Second {
					delete(e.peers, id)
				}
			}
			e.peerLock.Unlock()
		}
	}
}
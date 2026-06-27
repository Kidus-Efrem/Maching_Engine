package network

import (
	"encoding/binary"
	"log"
	"net"
	"sync"
)

// MarketDataPacketSize defines our strict 17-byte layout boundary
const MarketDataPacketSize = 17

type MarketDataUpdate struct {
	Symbol   [4]byte
	BidPrice uint64
	AskPrice uint32
}

type MarketDataHub struct {
	listenAddr string
	mu         sync.RWMutex
	clients    map[string]net.Conn
	inputChan  chan MarketDataUpdate
}

func NewMarketDataHub(listenAddr string) *MarketDataHub {
	return &MarketDataHub{
		listenAddr: listenAddr,
		clients:    make(map[string]net.Conn),
		inputChan:  make(chan MarketDataUpdate, 10000), // Generous buffer depth
	}
}

// Start opens the socket and fires off the async background broadcast multiplexer
func (h *MarketDataHub) Start() error {
	listener, err := net.Listen("tcp", h.listenAddr)
	if err != nil {
		return err
	}

	log.Printf("[PUB] Market Data Feed listening on %s\n", h.listenAddr)

	// Spin up the single background thread dedicated to fanning out packets
	go h.broadcastLoop()

	// Accept inbound subscribers concurrently
	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("[PUB] Subscription reject error: %v\n", err)
				continue
			}

			h.mu.Lock()
			h.clients[conn.RemoteAddr().String()] = conn
			h.mu.Unlock()

			log.Printf("[PUB] New subscriber connected: %s\n", conn.RemoteAddr().String())
		}
	}()

	return nil
}

// Publish offers an update to the channel without blocking the caller frame
func (h *MarketDataHub) Publish(update MarketDataUpdate) {
	select {
	case h.inputChan <- update:
	default:
		// Drop packet if the internal hub queue maxes out to safeguard memory stability
		log.Println("[PUB] Warning: Market data internal queue full. Dropping update frame.")
	}
}

func (h *MarketDataHub) broadcastLoop() {
	buf := make([]byte, MarketDataPacketSize)
	buf[0] = 'B' // Fixed MsgType for BBO (Best Bid/Offer) updates

	for update := range h.inputChan {
		// Serialize structural data directly into our reusable byte buffer slice
		copy(buf[1:5], update.Symbol[:])
		binary.BigEndian.PutUint64(buf[5:13], update.BidPrice)
		binary.BigEndian.PutUint32(buf[13:17], update.AskPrice)

		h.mu.RLock()
		for addr, conn := range h.clients {
			_, err := conn.Write(buf)
			if err != nil {
				log.Printf("[PUB] Dropping unresponsive subscriber %s: %v\n", addr, err)
				conn.Close()

				// Clean up broken connection safely outside RLock or handle via map cleanup pool
				h.mu.RUnlock()
				h.mu.Lock()
				delete(h.clients, addr)
				h.mu.Unlock()
				h.mu.RLock()
			}
		}
		h.mu.RUnlock()
	}
}
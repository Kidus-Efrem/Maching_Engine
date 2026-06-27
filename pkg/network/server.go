package network

import (
	// "binary"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"matching-engine/pkg/engine"
	"net"
)

// PacketSize defines our strict 33-byte protocol layout boundary
const PacketSize = 34 // 1 + 8 + 8 + 4 + 8 + 4 + 1

type TCPServer struct {
	listenAddr string
	ring       *engine.RingBuffer
}

func NewTCPServer(listenAddr string, ring *engine.RingBuffer) *TCPServer {
	return &TCPServer{
		listenAddr: listenAddr,
		ring:       ring,
	}
}

// UnmarshalOrder decodes a raw 34-byte slice directly into an Engine Order struct
// without causing any dynamic memory allocations on the heap.
func UnmarshalOrder(data []byte) (engine.Order, uint64, error) {
	if len(data) < PacketSize {
		return engine.Order{}, 0, errors.New("malformed packet: insufficient byte length")
	}

	// Offset 0: MsgType (e.g., 'O' for Order)
	msgType := data[0]
	if msgType != 'O' {
		return engine.Order{}, 0, errors.New("unsupported message type")
	}

	// Offset 1-8: OrderID (uint64)
	orderID := binary.BigEndian.Uint64(data[1:9])

	// Offset 9-16: AccountID (uint64)
	accountID := binary.BigEndian.Uint64(data[9:17])

	// Offset 17-20: Symbol ([4]byte)
	var symbol [4]byte
	copy(symbol[:], data[17:21])

	// Offset 21-28: Price (uint64)
	price := binary.BigEndian.Uint64(data[21:29])

	// Offset 29-32: Quantity (uint32)
	quantity := binary.BigEndian.Uint32(data[29:33])

	// Offset 33: IsBuy (bool represented as a single byte 1 or 0)
	isBuy := data[33] == 1

	// Construct our stack-allocated value order footprint
	order := engine.Order{
		ID:       orderID,
		Symbol:   symbol,
		Price:    price,
		Quantity: quantity,
		IsBuy:    isBuy,
	}

	return order, accountID, nil
}

// Start opens the socket gateway and kicks off the concurrent connection accept loop
func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("[SERVER] FIX-lite TCP Gateway humming on %s\n", s.listenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[SERVER] Connection drop error: %v\n", err)
			continue
		}

		// Handle each connection concurrently in its own green-thread context
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("[SERVER] Client connected: %s\n", conn.RemoteAddr().String())

	// A fixed-width reusable buffer frame for this thread's connection lifecycle
	buf := make([]byte, PacketSize)

	for {
		// ReadFull guarantees we block until we pull exactly 34 bytes off the wire
		_, err := io.ReadFull(conn, buf)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				log.Printf("[SERVER] Client disconnected: %s\n", conn.RemoteAddr().String())
				return
			}
			log.Printf("[SERVER] Read stream failure: %v\n", err)
			return
		}

		// Decode the packet without allocation
		order, accountID, err := UnmarshalOrder(buf)
		if err != nil {
			log.Printf("[SERVER] Protocol violation: %v\n", err)
			continue // Drain packet bad signature, keep connection alive
		}

		// Fire it directly down the atomic ingestion channel pipeline!
		s.ring.Enqueue(order, accountID)
	}
}
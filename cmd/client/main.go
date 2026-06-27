package main

import (
	"encoding/binary"
	"log"
	"net"
	"time"
)

const PacketSize = 34
const ServerAddr = "127.0.0.1:8080"

func main() {
	log.Printf("[CLIENT] Connecting to Matching Engine at %s...\n", ServerAddr)
	conn, err := net.Dial("tcp", ServerAddr)
	if err != nil {
		log.Fatalf("[CLIENT] Connection failed: %v\n", err)
	}
	defer conn.Close()
	log.Println("[CLIENT] Connected! Blasting 5 mock orders...")

	// Fixed 4-byte ticker symbol row
	symbol := [4]byte{'A', 'A', 'P', 'L'}
	buf := make([]byte, PacketSize)

	// Blast 5 sequential alternating orders
	for i := uint64(1); i <= 5; i++ {
		isBuy := byte(1) // BUY
		if i%2 == 0 {
			isBuy = byte(0) // SELL
		}

		// Pack fields according to our strict byte offset mapping specifications
		buf[0] = 'O'                                 // MsgType
		binary.BigEndian.PutUint64(buf[1:9], i)       // OrderID
		binary.BigEndian.PutUint64(buf[9:17], 7)     // AccountID (Our default provisioned account)
		copy(buf[17:21], symbol[:])                   // Symbol
		binary.BigEndian.PutUint64(buf[21:29], 15000) // Price ($150.00)
		binary.BigEndian.PutUint32(buf[29:33], 10)    // Quantity
		buf[33] = isBuy                              // IsBuy flag

		_, err := conn.Write(buf)
		if err != nil {
			log.Fatalf("[CLIENT] Failed to write to stream socket: %v\n", err)
		}
		log.Printf("[CLIENT] Sent Order #%d (IsBuy=%d)\n", i, isBuy)
		time.Sleep(100 * time.Millisecond) // Short delay so we can see the logs tick
	}

	log.Println("[CLIENT] Done bursting initial batch. Disconnecting.")
}
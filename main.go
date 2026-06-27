package main

import (
	"log"
	"matching-engine/pkg/engine"
	"matching-engine/pkg/network"
)

func main() {
	log.Println("[MAIN] Initializing High-Velocity Matching Engine platform...")

	// 1. Initialize core sub-systems
	riskGate := engine.NewRiskEngine()
	exchange := engine.NewExchange(nil, riskGate) // WAL set to nil for baseline testing
	ring := engine.NewRingBuffer()

	// 2. Provision a default trading account profile for network baseline testing
	defaultTraderID := uint64(7)
	riskGate.Accounts[defaultTraderID] = &engine.AccountState{
		AccountID:         defaultTraderID,
		AvailableBalance:  1_000_000_000, // Massive cash runway
		ActiveMaxPosition: 1_000_000,     // Deep position headroom
	}
	log.Printf("[MAIN] Default trader account #%d provisioned safely\n", defaultTraderID)

	// 3. Fire up the background single-threaded matching core engine loop
	ring.StartConsuming(exchange)
	log.Println("[MAIN] Lock-free consumer pipeline engine online")

	// 4. Instantiated and bind the FIX-lite TCP Gateway
	serverAddress := "127.0.0.1:8080"
	server := network.NewTCPServer(serverAddress, ring)

	// 5. Block main execution loop and handle network traffic
	log.Printf("[MAIN] Handing off control flow to TCP network layer\n")
	if err := server.Start(); err != nil {
		log.Fatalf("[MAIN] Critical engine failure during runtime boot: %v\n", err)
	}
}
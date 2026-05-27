package main

import (
	"fmt"
	"matching-engine/pkg/engine"
	"os"
)

func main() {
	snapshotPath := "exchange_global.snapshot"

	aapl := [4]byte{'A', 'A', 'P', 'L'}
	msft := [4]byte{'M', 'S', 'F', 'T'}
	traderID := uint64(77)

	fmt.Println("🌀 STEP 1: Bootstrapping live exchange with multiple distinct ticker rooms...")
	liveExchange := engine.NewExchange(nil, nil)

	// Injecting isolated liquidity across independent assets
	liveExchange.ProcessOrder(&engine.Order{ID: 101, Symbol: aapl, Price: 15000, Quantity: 10, IsBuy: false}, traderID) // AAPL Ask at $150
	liveExchange.ProcessOrder(&engine.Order{ID: 201, Symbol: msft, Price: 28000, Quantity: 35, IsBuy: false}, traderID) // MSFT Ask at $280

	fmt.Printf("Live AAPL Best Ask: $%d.00\n", liveExchange.Books[aapl].GetBestAsk()/100)
	fmt.Printf("Live MSFT Best Ask: $%d.00\n", liveExchange.Books[msft].GetBestAsk()/100)

	fmt.Println("\n💾 STEP 2: Freezing the entire multi-asset exchange state into a single global snapshot...")
	snap := engine.NewSnapshotEngine(snapshotPath)
	if err := snap.SaveExchangeSnapshot(liveExchange); err != nil {
		panic(err)
	}

	// 🚨 DETONATE SYSTEM RAM VARIABLES
	fmt.Println("💥 [CRASH EVENT]: Vaporizing active memory states...")
	liveExchange = nil

	fmt.Println("\n🔄 STEP 3: Initializing system reboot. Rehydrating from global multi-asset snapshot file...")
	recoveredExchange, err := snap.LoadExchangeSnapshot()
	if err != nil {
		panic(err)
	}

	fmt.Println("\n✅ STEP 4: Running isolated verification audit loops on separate asset rooms:")
	fmt.Printf("Recovered AAPL Best Ask State: $%d.00 (Should be $150.00)\n", recoveredExchange.Books[aapl].GetBestAsk()/100)
	fmt.Printf("Recovered MSFT Best Ask State: $%d.00 (Should be $280.00)\n", recoveredExchange.Books[msft].GetBestAsk()/100)

	// Clean up after test run
	os.Remove(snapshotPath)
}
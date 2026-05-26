package main

import (
	"fmt"
	"matching-engine/pkg/engine"
	"os"
)

func main() {
	snapshotPath := "engine.snapshot"
	aapl := [4]byte{'A', 'A', 'P', 'L'}

	fmt.Println("🌀 STEP 1: Bootstrapping live market book state...")
	liveBook := engine.NewOrderBook()

	// Seed resting state values
	liveBook.ProcessOrder(&engine.Order{ID: 801, Symbol: aapl, Price: 10500, Quantity: 100, IsBuy: false})
	liveBook.ProcessOrder(&engine.Order{ID: 802, Symbol: aapl, Price: 10600, Quantity: 200, IsBuy: false})
	fmt.Printf("Current Live Market Ask Boundary: $%d.00\n", liveBook.GetBestAsk()/100)

	fmt.Println("\n💾 STEP 2: Freezing live memory structures to binary snapshot file...")
	snap := engine.NewSnapshotEngine(snapshotPath)
	if err := snap.SaveSnapshot(liveBook); err != nil {
		panic(err)
	}

	// 🚨 DETONATE MEMORY STATE (SIMULATE HARD BLACKOUT)
	fmt.Println("💥 [CRASH ]: Purging all application variables from RAM...")
	liveBook = nil

	fmt.Println("\n🔄 STEP 3: Initializing system reboot. Rehydrating from frozen snapshot records...")
	recoveredBook, err := snap.LoadSnapshot()
	if err != nil {
		panic(err)
	}

	fmt.Println("\n✅ STEP 4: Running verification integrity check on recovered memory state:")
	fmt.Printf("Recovered Book Best Ask Price: $%d.00 (Should be $105.00)\n", recoveredBook.GetBestAsk()/100)

	// Check queue length depth to ensure both elements recovered safely
	fmt.Printf("Total Recovered Tracking Entries: %d items (Should be 2)\n", len(recoveredBook.OrdersRegistry))

	// Clean up snapshot file from directory
	os.Remove(snapshotPath)
}
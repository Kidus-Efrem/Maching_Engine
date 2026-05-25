package main

import (
	"fmt"
	"matching-engine/pkg/engine"
	"os"
)

func main() {
	walPath := "engine.wal"

	fmt.Println("⚡ CRASH SIMULATOR INITIALIZED...")
	fmt.Println("Step 1: Instantiating fresh log file and logging active liquidity...")

	wal, err := engine.OpenWAL(walPath)
	if err != nil {
		panic(err)
	}

	// Start exchange phase
	exchange := engine.NewExchange(wal)
	aapl := [4]byte{'A', 'A', 'P', 'L'}

	// Place two resting sell orders onto the disk and RAM layers
	exchange.ProcessOrder(&engine.Order{ID: 701, Symbol: aapl, Price: 15000, Quantity: 50, IsBuy: false})
	exchange.ProcessOrder(&engine.Order{ID: 702, Symbol: aapl, Price: 15200, Quantity: 30, IsBuy: false})

	fmt.Printf("Initial Live Market Ask State: $%d.00\n", exchange.Books[aapl].GetBestAsk()/100)
	wal.Close()

	// 🚨 SIMULATE CATASTROPHIC ENGINE CRASH
	fmt.Println("\n💥 [CRASH EVENT]: Server power cords pulled! Wiping RAM states...")
	exchange = nil // Pure memory vaporization!

	// 🛠️ REBOOT AND RECOVERY STAGE
	fmt.Println("\n🔄 [REBOOT PHASE]: Spinning system backup. Reading binary WAL states from physical disk...")
	recoveryWal, err := engine.OpenWAL(walPath)
	if err != nil {
		panic(err)
	}
	defer recoveryWal.Close()

	// Read raw bits off the disk file
	historicalOrders, err := recoveryWal.ReadAll()
	if err != nil {
		panic(err)
	}

	// Rehydrate a pristine independent engine
	rebootedExchange := engine.NewExchange(nil) // Pass nil so it doesn't infinite loop write back to itself

	// Replay history
	for _, historicalOrder := range historicalOrders {
		rebootedExchange.ProcessOrder(historicalOrder)
		fmt.Printf("REPLAYED LOG ENTRY: Rehydrated Order ID %d for Ticker %s @ $%d.00\n",
			historicalOrder.ID, string(historicalOrder.Symbol[:]), historicalOrder.Price/100)
	}

	fmt.Println("\n✅ RECOVERY AUDIT COMPLETION SUCCESSFUL:")
	fmt.Printf("Recovered Engine Active Best Ask State: $%d.00\n", rebootedExchange.Books[aapl].GetBestAsk()/100)

	// Clean up after our test run completes
	os.Remove(walPath)
}
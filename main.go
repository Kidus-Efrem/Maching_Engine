package main

import (
	"fmt"
	"matching-engine/pkg/engine"
)

func main() {
	// Initialize our multi-asset exchange supervisor
	exchange := engine.NewExchange()

	aapl := [4]byte{'A', 'A', 'P', 'L'}
	msft := [4]byte{'M', 'S', 'F', 'T'}

	fmt.Println("🚀 Loading market data with overlapping prices across different symbols...")

	// 1. Place a resting Sell Order for AAPL at $150
	exchange.ProcessOrder(&engine.Order{ID: 1, Symbol: aapl, Price: 15000, Quantity: 10, IsBuy: false})

	// 2. Place a resting Sell Order for MSFT at the EXACT SAME price ($150)
	exchange.ProcessOrder(&engine.Order{ID: 2, Symbol: msft, Price: 15000, Quantity: 20, IsBuy: false})

	fmt.Println("⚡ Firing an aggressive Buyer who wants AAPL at $150...")

	// 3. Buyer wants AAPL at $150. This should ONLY match with Order 1 (AAPL), completely ignoring Order 2 (MSFT).
	buyerOrder := &engine.Order{
		ID:       999,
		Symbol:   aapl,
		Price:    15000,
		Quantity: 10,
		IsBuy:    true,
	}

	trades := exchange.ProcessOrder(buyerOrder)

	fmt.Printf("\n--- MATCHING RESULTS FOR %s ---\n", string(buyerOrder.Symbol[:]))
	if len(trades) == 0 {
		fmt.Println("No matches found.")
	}
	for _, t := range trades {
		fmt.Printf("SUCCESS: Matched Buy %d with Sell %d | Qty: %d @ $%d for Asset: %s\n",
			t.BuyOrderID, t.SellOrderID, t.Quantity, t.Price/100, string(t.Symbol[:]))
	}

	// Verify MSFT book still contains its resting seller intact
	msftBook := exchange.Books[msft]
	fmt.Printf("\n--- SAFETY SANITY CHECK ---\n")
	fmt.Printf("MSFT Active Best Ask Price: $%d (Should still be $150)\n", msftBook.AskHeap.prices[0]/100)
}
package main

import (
	"fmt"
	"matching-engine/pkg/engine"
)

func main() {
	exchange := engine.NewExchange(nil, nil)
	aapl := [4]byte{'A', 'A', 'P', 'L'}
	traderID := uint64(10)

	fmt.Println("🚀 Queueing 3 resting Sell Orders for AAPL at $100.00...")
	exchange.ProcessOrder(&engine.Order{ID: 501, Symbol: aapl, Price: 10000, Quantity: 10, IsBuy: false}, traderID)
	exchange.ProcessOrder(&engine.Order{ID: 502, Symbol: aapl, Price: 10000, Quantity: 20, IsBuy: false}, traderID) // The Middle Order
	exchange.ProcessOrder(&engine.Order{ID: 503, Symbol: aapl, Price: 10000, Quantity: 15, IsBuy: false}, traderID)

	fmt.Println("⚡ Issuing an instantaneous O(1) Cancellation Request for Order #502...")
	err := exchange.CancelOrder(aapl, 502)
	if err != nil {
		fmt.Printf("❌ Cancellation Failed: %v\n", err)
	} else {
		fmt.Println("✅ Success: Order #502 completely unlinked and wiped from memory registry!")
	}

	fmt.Println("\n💥 Firing an aggressive Buyer for 25 shares to cross the market at $100.00...")
	// Buyer wants 25 shares. It should fill Order 501 (10 shares), skip 502 completely, and take the final 15 shares from 503!
	trades, _ := exchange.ProcessOrder(&engine.Order{ID: 999, Symbol: aapl, Price: 10000, Quantity: 25, IsBuy: true}, traderID)

	fmt.Printf("\n--- MATCHING RESULTS POST-CANCELLATION ---\n")
	for _, t := range trades {
		fmt.Printf("MATCHED: Buy Order %d matched Sell Order %d | Qty: %d shares @ $%d.00\n",
			t.BuyOrderID, t.SellOrderID, t.Quantity, t.Price/100)
	}
}
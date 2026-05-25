package main

import (
	"fmt"
	"matching-engine/pkg/engine"
)

func main() {
	book := engine.NewOrderBook()
	symbol := [4]byte{'A', 'A', 'P', 'L'}

	fmt.Println("🚀 Stage 1: Loading Order Book shelves with resting Sell Orders...")

	// Person A selling 10 shares at $101
	book.ProcessOrder(&engine.Order{ID: 101, Symbol: symbol, Price: 10100, Quantity: 10, IsBuy: false})
	// Person B selling 20 shares at $101 (sits behind Person A on the same shelf)
	book.ProcessOrder(&engine.Order{ID: 102, Symbol: symbol, Price: 10100, Quantity: 20, IsBuy: false})
	// Person C selling 15 shares at $102 (higher shelf)
	book.ProcessOrder(&engine.Order{ID: 103, Symbol: symbol, Price: 10200, Quantity: 15, IsBuy: false})

	fmt.Printf("Current Market Spread: Best Bid: $%d | Best Ask: $%d\n\n", book.BestBid/100, book.BestAsk/100)

	fmt.Println("⚡ Stage 2: Introducing aggressive Inbound Buy Order for 40 shares up to $105...")
	aggressiveBuy := &engine.Order{
		ID:       501,
		Symbol:   symbol,
		Price:    10500, // Willing to pay up to $105
		Quantity: 40,    // Wants 40 shares
		IsBuy:    true,
	}

	trades := book.ProcessOrder(aggressiveBuy)

	// Display trade match results
	for _, t := range trades {
		fmt.Printf("MATCHED TRADE: Buy Order %d matched Sell Order %d | Qty: %d @ $%d\n",
			t.BuyOrderID, t.SellOrderID, t.Quantity, t.Price/100)
	}

	fmt.Printf("\nPost-Match Market Spread: Best Bid: $%d | Best Ask: $%d\n", book.BestBid/100, book.BestAsk/100)
	fmt.Printf("Aggressive Buy Order Remainder Leaves Unfilled Shares: %d\n", aggressiveBuy.Quantity)
}
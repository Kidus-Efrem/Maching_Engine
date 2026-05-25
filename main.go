package main

import (
	"fmt"
	"matching-engine/pkg/engine"
)

func main() {
	// Initialize our top-level multi-asset exchange supervisor
	exchange := engine.NewExchange()

	// Define explicit fixed 4-byte identifiers for our assets
	aapl := [4]byte{'A', 'A', 'P', 'L'}
	msft := [4]byte{'M', 'S', 'F', 'T'}

	fmt.Println("============ STAGE 1: INJECTING MARKET LIQUIDITY ============")
	fmt.Println("Populating separate asset books with resting limit sell orders...")

	exchange.ProcessOrder(&engine.Order{ID: 101, Symbol: aapl, Price: 10100, Quantity: 10, IsBuy: false})
	exchange.ProcessOrder(&engine.Order{ID: 102, Symbol: aapl, Price: 10100, Quantity: 20, IsBuy: false})
	exchange.ProcessOrder(&engine.Order{ID: 103, Symbol: aapl, Price: 10200, Quantity: 15, IsBuy: false})
	exchange.ProcessOrder(&engine.Order{ID: 201, Symbol: msft, Price: 10100, Quantity: 25, IsBuy: false})

	aaplBook := exchange.Books[aapl]
	msftBook := exchange.Books[msft]

	// FIXED: Calling the exported public getter methods
	fmt.Printf("\n[Current Spread] AAPL Best Ask Price: $%d.00\n", aaplBook.GetBestAsk()/100)
	fmt.Printf("[Current Spread] MSFT Best Ask Price: $%d.00\n\n", msftBook.GetBestAsk()/100)

	fmt.Println("============ STAGE 2: EXECUTING MULTI-SHELF CROSSES ============")
	fmt.Println("Firing an aggressive Buy Order for 40 shares of AAPL up to a max price of $105.00...")

	aggressiveBuy := &engine.Order{
		ID:       501,
		Symbol:   aapl,
		Price:    10500,
		Quantity: 40,
		IsBuy:    true,
	}

	trades := exchange.ProcessOrder(aggressiveBuy)

	fmt.Printf("\n--- MATCHING RESULTS FOR TRANSACTION RUN ---\n")
	if len(trades) == 0 {
		fmt.Println("No matches executed.")
	}
	for _, t := range trades {
		fmt.Printf("MATCH EXECUTED: Buy Order %d matched Sell Order %d | Volume: %d shares @ $%d.00 for Asset: %s\n",
			t.BuyOrderID, t.SellOrderID, t.Quantity, t.Price/100, string(t.Symbol[:]))
	}

	fmt.Println("\n============ STAGE 3: POST-MATCH AUDIT SANITY CHECK ============")

	// FIXED: Calling the exported public getter methods
	fmt.Printf("AAPL Next Best Ask Price remaining: $%d.00 (Should be $102.00)\n", aaplBook.GetBestAsk()/100)
	fmt.Printf("AAPL Leftover Resting Buyer Shares: %d (Should be 0, fully consumed)\n", aggressiveBuy.Quantity)

	fmt.Printf("MSFT Ask Level Count: %d shares remaining (Should be 25, completely isolated!)\n", msftBook.Asks[10100].Head.Value.Quantity)
}
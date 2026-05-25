package main

import (
	"fmt"
	"matching-engine/pkg/engine"
)

func main() {
	// Initialize our Risk Engine and set up a trader profile
	riskGate := engine.NewRiskEngine()
	traderAccountID := uint64(42)

	riskGate.Accounts[traderAccountID] = &engine.AccountState{
		AccountID:         traderAccountID,
		AvailableBalance:  500000, // $5,000.00 scaled cash capital
		ActiveMaxPosition: 1000,   // Cannot buy/sell more than 1,000 shares at once
	}

	// Initialize Exchange passing our Risk gatekeeper
	exchange := engine.NewExchange(nil, riskGate)
	aapl := [4]byte{'A', 'A', 'P', 'L'}

	fmt.Println("🚀 Case 1: Submitting a valid order within the trader's budget limits...")
	validOrder := &engine.Order{ID: 1, Symbol: aapl, Price: 10000, Quantity: 10, IsBuy: true} // Cost: $1,000.00

	_, err := exchange.ProcessOrder(validOrder, traderAccountID)
	if err != nil {
		fmt.Printf("❌ Unexpected Failure: %v\n", err)
	} else {
		fmt.Println("✅ Success: Order passed risk validation and was queued into the matching engine!")
	}

	fmt.Println("\n⚡ Case 2: Submitting an invalid order that exceeds available credit balance...")
	expensiveOrder := &engine.Order{ID: 2, Symbol: aapl, Price: 10000, Quantity: 60, IsBuy: true} // Cost: $6,000.00 (Exceeds $5k balance)

	_, err = exchange.ProcessOrder(expensiveOrder, traderAccountID)
	if err != nil {
		fmt.Printf("🛡️ RISK INTERCEPTION: %v (Successfully Blocked!)\n", err)
	} else {
		fmt.Println("❌ Failure: System allowed an illegal over-leveraged trade to pass through!")
	}
}
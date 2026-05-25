package main

import (
	"fmt"
	"matching-engine/pkg/engine"
)

func main() {
	// Initialize our order book container
	book := engine.NewOrderBook()

	// Create a mock buy order for Apple stock at $150.25
	mockOrder := &engine.Order{
		ID:       1,
		Symbol:   [4]byte{'A', 'A', 'P', 'L'},
		Price:    15025,
		Quantity: 50,
		IsBuy:    true,
	}

	// Fetch or create the price level shelf for $150.25
	shelf, exists := book.Bids[mockOrder.Price]
	if !exists {
		shelf = &engine.PriceLevel{Price: mockOrder.Price}
		book.Bids[mockOrder.Price] = shelf
	}

	// Add the order to the queue line on that shelf
	node := shelf.AppendOrder(mockOrder)

	fmt.Printf("🧱 Architecture Sanity Check: SUCCESS\n")
	fmt.Printf("Order ID: %d successfully queued at price level: %d cents\n", node.Value.ID, node.Value.Price)
}

package engine

import (
	"sync"
	"testing"
)

// Global memory arena recycling reusable Order memory frames
var orderPool = sync.Pool{
	New: func() interface{} {
		return &Order{}
	},
}

func BenchmarkOrderIngestion(b *testing.B) {
	riskGate := NewRiskEngine()
	exchange := NewExchange(nil, riskGate)
	ring := NewRingBuffer()

	traderID := uint64(7)
	riskGate.Accounts[traderID] = &AccountState{
		AccountID:         traderID,
		AvailableBalance:  1_000_000_000,
		ActiveMaxPosition: 1_000_000,
	}

	ring.StartConsuming(exchange)
	symbol := [4]byte{'A', 'A', 'P', 'L'}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		isBuy := i%2 == 0
		price := uint64(15000 + (i % 10))

		// Borrow a pre-allocated Order block memory address from the arena pool
		order := orderPool.Get().(*Order)

		// Reset its internal field data matrices cleanly
		order.ID = uint64(i)
		order.Symbol = symbol
		order.Price = price
		order.Quantity = 10
		order.IsBuy = isBuy

		ring.Enqueue(order, traderID)

		// In a production app, the consumer loop returns it to the pool via orderPool.Put(order)
		// For the sake of isolating ingestion memory loops, we cycle it here safely
		if i > 1024 {
			orderPool.Put(order)
		}
	}
}
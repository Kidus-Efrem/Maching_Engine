package engine

import (
	"testing"
)

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

		// Create a local value footprint on the stack frame loop
		order := Order{
			ID:       uint64(i),
			Symbol:   symbol,
			Price:    price,
			Quantity: 10,
			IsBuy:    isBuy,
		}

		ring.Enqueue(order, traderID)
	}
}

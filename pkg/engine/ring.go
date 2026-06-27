package engine

import (
	"runtime"
	"sync/atomic"
)

const BufferSize = 1024
const BufferMask = BufferSize - 1

type RingEntry struct {
	Order     Order // Storing a flat value copy ensures absolute cross-thread isolation
	AccountID uint64
}

type RingBuffer struct {
	buffer           [BufferSize]RingEntry
	_                [56]byte // Cache line padding to prevent false sharing
	producerSequence uint64
	_                [56]byte // Cache line padding to prevent false sharing
	consumerSequence uint64
}

func NewRingBuffer() *RingBuffer {
	return &RingBuffer{
		producerSequence: 0,
		consumerSequence: 0,
	}
}

// Enqueue accepts a flat Order struct copy by value to guarantee thread safety
func (rb *RingBuffer) Enqueue(order Order, accountID uint64) {
	for {
		currentProducerSeq := atomic.LoadUint64(&rb.producerSequence)
		currentConsumerSeq := atomic.LoadUint64(&rb.consumerSequence)

		if currentProducerSeq-currentConsumerSeq >= BufferSize {
			runtime.Gosched() // Buffer full, yield thread execution context
			continue
		}

		if atomic.CompareAndSwapUint64(&rb.producerSequence, currentProducerSeq, currentProducerSeq+1) {
			index := currentProducerSeq & BufferMask
			rb.buffer[index] = RingEntry{Order: order, AccountID: accountID}
			return
		}
	}
}

func (rb *RingBuffer) StartConsuming(exchange *Exchange) {
	go func() {
		for {
			currentConsumerSeq := rb.consumerSequence
			currentProducerSeq := atomic.LoadUint64(&rb.producerSequence)

			if currentConsumerSeq < currentProducerSeq {
				index := currentConsumerSeq & BufferMask
				entry := rb.buffer[index]

				if entry.Order.Quantity > 0 {
					// Pass a localized pointer down to the single-threaded matching core
					_, _ = exchange.ProcessOrder(&entry.Order, entry.AccountID)
					rb.buffer[index] = RingEntry{}
					rb.consumerSequence++
				}
			} else {
				runtime.Gosched() // Buffer empty, yield thread execution context
			}
		}
	}()
}
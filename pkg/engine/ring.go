package engine

import (
	"runtime"
	"sync/atomic"
)

const BufferSize = 1024
const BufferMask = BufferSize - 1

type RingEntry struct {
	Order     *Order
	AccountID uint64
}

type RingBuffer struct {
	buffer           [BufferSize]RingEntry
	producerSequence uint64
	consumerSequence uint64
}

func NewRingBuffer() *RingBuffer {
	return &RingBuffer{
		producerSequence: 0,
		consumerSequence: 0,
	}
}

func (rb *RingBuffer) Enqueue(order *Order, accountID uint64) {
	for {
		currentProducerSeq := atomic.LoadUint64(&rb.producerSequence)
		currentConsumerSeq := atomic.LoadUint64(&rb.consumerSequence)

		if currentProducerSeq-currentConsumerSeq >= BufferSize {
			runtime.Gosched()
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

				if entry.Order != nil {
					// We discard the outputs cleanly here using blank identifiers
					_, _ = exchange.ProcessOrder(entry.Order, entry.AccountID)
					rb.buffer[index] = RingEntry{}
					rb.consumerSequence++
				}
			} else {
				runtime.Gosched()
			}
		}
	}()
}
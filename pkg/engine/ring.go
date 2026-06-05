package engine

import (
	"runtime"
	"sync/atomic"
)

// Define our buffer capacity as a strict power of 2 (1024) to enable bitwise AND wrap-around math.
const BufferSize = 1024
const BufferMask = BufferSize - 1

// RingEntry acts as a single data slot wrapper inside the circular queue.
type RingEntry struct {
	Order     *Order
	AccountID uint64
}

// RingBuffer manages thread-safe, lock-free coordination between network inputs and the engine core.
type RingBuffer struct {
	buffer           [BufferSize]RingEntry
	producerSequence uint64 // Tracks total items written to the buffer
	consumerSequence uint64 // Tracks total items read by the matching engine
}

// NewRingBuffer initializes our atomic sequence tracker indices.
func NewRingBuffer() *RingBuffer {
	return &RingBuffer{
		producerSequence: 0,
		consumerSequence: 0,
	}
}

// Enqueue is called by multi-threaded network producers to drop an order into the tracking circle.
func (rb *RingBuffer) Enqueue(order *Order, accountID uint64) {
	for {
		currentProducerSeq := atomic.LoadUint64(&rb.producerSequence)
		currentConsumerSeq := atomic.LoadUint64(&rb.consumerSequence)

		// Guard: Check if the circular queue is completely full
		if currentProducerSeq-currentConsumerSeq >= BufferSize {
			runtime.Gosched() // Buffer is backed up; yield CPU thread slice and retry
			continue
		}

		// Attempt to claim this sequence slot atomically via Compare-And-Swap (CAS)
		if atomic.CompareAndSwapUint64(&rb.producerSequence, currentProducerSeq, currentProducerSeq+1) {
			// Calculate our fixed index using the bitwise AND mask optimization instead of slow modulo (%) division
			index := currentProducerSeq & BufferMask

			// Store the transaction entry directly into the allocated memory array slot
			rb.buffer[index] = RingEntry{Order: order, AccountID: accountID}
			return
		}
	}
}

// StartConsuming fires off the single-threaded infinite loop processing engine.
func (rb *RingBuffer) StartConsuming(exchange *Exchange) {
	// Run the engine worker inside an independent, single background thread context
	go func() {
		for {
			currentConsumerSeq := rb.consumerSequence
			currentProducerSeq := atomic.LoadUint64(&rb.producerSequence)

			// Check if there is an unread order waiting in the queue
			if currentConsumerSeq < currentProducerSeq {
				index := currentConsumerSeq & BufferMask
				entry := rb.buffer[index]

				// Safety check: Ensure the producer has finished writing data to this slot
				if entry.Order != nil {
					// ROUTE DIRECTLY INTO OUR SAFE SINGLE-THREADED ENGINE PIPELINE!
					exchange.ProcessOrder(entry.Order, entry.AccountID)

					// Wipe the slot clean to avoid stale memory references
					rb.buffer[index] = RingEntry{}

					// Advance our consumer sequence head pointer forward sequentially
					rb.consumerSequence++
				}
			} else {
				// The ring buffer is completely caught up.
				// Yield the CPU thread to prevent core overheating during low-volume periods.
				runtime.Gosched()
			}
		}
	}()
}

package engine

import (
	"container/heap"
)

// ============================================================================
// 1. HARDWARE-OPTIMIZED DATA MODELS
// ============================================================================

// Order represents a user's request to buy or sell a specific security.
type Order struct {
	ID       uint64  // Unique identifier for the order
	Symbol   [4]byte // Fixed 4-byte array (e.g., 'A','A','P','L') preventing heap-allocation
	Price    uint64  // Scaled integer representation (e.g., $100.53 is 10053) to eliminate float rounding errors
	Quantity uint32  // Number of shares requested
	IsBuy    bool    // True for Buy (Bid), False for Sell (Ask)
}

// OrderNode wraps our Order struct inside an intrusive Doubly Linked List node.
type OrderNode struct {
	Value *Order     // Pointer to the underlying order data
	Prev  *OrderNode // Pointer to the order ahead of this one in the queue line
	Next  *OrderNode // Pointer to the order behind this one in the queue line
}

// PriceLevel represents a single unified "price shelf" containing a FIFO queue.
type PriceLevel struct {
	Price uint64     // The price value this specific level tracks
	Head  *OrderNode // The oldest resting order (front of the line, matches first)
	Tail  *OrderNode // The newest resting order (back of the line)
}

// AppendOrder places an incoming order at the absolute tail of this shelf line.
func (pl *PriceLevel) AppendOrder(order *Order) *OrderNode {
	newNode := &OrderNode{Value: order}

	if pl.Head == nil {
		pl.Head = newNode
		pl.Tail = newNode
	} else {
		pl.Tail.Next = newNode
		newNode.Prev = pl.Tail
		pl.Tail = newNode
	}
	return newNode
}

// TradeExecution represents a finalized binding match between a buyer and a seller.
type TradeExecution struct {
	BuyOrderID  uint64   // Tracking ID of the buyer
	SellOrderID uint64   // Tracking ID of the seller
	Price       uint64   // The transaction clearance price
	Quantity    uint32   // The volume of shares swapped
	Symbol      [4]byte  // The asset symbol frame
}

// ============================================================================
// 2. PRIORITY BOUNDARY PRICE HEAP
// ============================================================================

// PriceHeap implements the container/heap interface to manage active price keys.
type PriceHeap struct {
	prices []uint64
	isMin  bool // True for Asks (lowest price priority), False for Bids (highest price priority)
}

func (h *PriceHeap) Len() int           { return len(h.prices) }
func (h *PriceHeap) Swap(i, j int)      { h.prices[i], h.prices[j] = h.prices[j], h.prices[i] }
func (h *PriceHeap) Push(x interface{}) { h.prices = append(h.prices, x.(uint64)) }
func (h *PriceHeap) Pop() interface{} {
	old := h.prices
	n := len(old)
	x := old[n-1]
	h.prices = old[0 : n-1]
	return x
}
func (h *PriceHeap) Less(i, j int) bool {
	if h.isMin {
		return h.prices[i] < h.prices[j] // Min-Heap logic
	}
	return h.prices[i] > h.prices[j] // Max-Heap logic
}

// ============================================================================
// 3. THE ISOLATED ORDER BOOK CONTAINER
// ============================================================================

// OrderBook orchestrates matching conditions for a single security asset.
type OrderBook struct {
	Bids    map[uint64]*PriceLevel // HashMap for O(1) direct bid level lookup
	Asks    map[uint64]*PriceLevel // HashMap for O(1) direct ask level lookup
	BidHeap *PriceHeap             // Max-Heap keeping the highest buy price at index 0
	AskHeap *PriceHeap             // Min-Heap keeping the lowest sell price at index 0
}

// NewOrderBook completely allocates memory for a pristine asset book space.
func NewOrderBook() *OrderBook {
	bidHeap := &PriceHeap{prices: make([]uint64, 0), isMin: false}
	askHeap := &PriceHeap{prices: make([]uint64, 0), isMin: true}
	heap.Init(bidHeap)
	heap.Init(askHeap)

	return &OrderBook{
		Bids:    make(map[uint64]*PriceLevel),
		Asks:    make(map[uint64]*PriceLevel),
		BidHeap: bidHeap,
		AskHeap: askHeap,
	}
}

// Internal unexported helper methods for inside the package
func (ob *OrderBook) getBestAsk() uint64 {
	if ob.AskHeap.Len() == 0 {
		return 0
	}
	return ob.AskHeap.prices[0]
}

func (ob *OrderBook) getBestBid() uint64 {
	if ob.BidHeap.Len() == 0 {
		return 0
	}
	return ob.BidHeap.prices[0]
}

// PUBLIC GATEWAY EXPORTED METHODS (Uppercase for cross-package access)

// GetBestAsk is the public gateway for outside packages to read the best ask.
func (ob *OrderBook) GetBestAsk() uint64 {
	return ob.getBestAsk()
}

// GetBestBid is the public gateway for outside packages to read the best bid.
func (ob *OrderBook) GetBestBid() uint64 {
	return ob.getBestBid()
}

// ============================================================================
// 4. PRICE-TIME ENGINE LOOP EXECUTION
// ============================================================================

// ProcessOrder scales through the opposing side's queue to fill an inbound transaction.
func (ob *OrderBook) ProcessOrder(incoming *Order) []*TradeExecution {
	var trades []*TradeExecution

	if incoming.IsBuy {
		for incoming.Quantity > 0 && ob.AskHeap.Len() > 0 && incoming.Price >= ob.getBestAsk() {
			bestAskPrice := ob.getBestAsk()
			askLevel := ob.Asks[bestAskPrice]
			currentNode := askLevel.Head

			for currentNode != nil && incoming.Quantity > 0 {
				restingOrder := currentNode.Value

				matchQty := incoming.Quantity
				if restingOrder.Quantity < matchQty {
					matchQty = restingOrder.Quantity
				}

				incoming.Quantity -= matchQty
				restingOrder.Quantity -= matchQty

				trades = append(trades, &TradeExecution{
					BuyOrderID:  incoming.ID,
					SellOrderID: restingOrder.ID,
					Price:       bestAskPrice,
					Quantity:    matchQty,
					Symbol:      incoming.Symbol,
				})

				if restingOrder.Quantity == 0 {
					askLevel.Head = currentNode.Next
					if askLevel.Head != nil {
						askLevel.Head.Prev = nil
					} else {
						askLevel.Tail = nil
					}
				}
				currentNode = askLevel.Head
			}

			if askLevel.Head == nil {
				delete(ob.Asks, bestAskPrice)
				heap.Pop(ob.AskHeap)
			}
		}

		if incoming.Quantity > 0 {
			shelf, exists := ob.Bids[incoming.Price]
			if !exists {
				shelf = &PriceLevel{Price: incoming.Price}
				ob.Bids[incoming.Price] = shelf
				heap.Push(ob.BidHeap, incoming.Price)
			}
			shelf.AppendOrder(incoming)
		}

	} else {
		for incoming.Quantity > 0 && ob.BidHeap.Len() > 0 && incoming.Price <= ob.getBestBid() {
			bestBidPrice := ob.getBestBid()
			bidLevel := ob.Bids[bestBidPrice]
			currentNode := bidLevel.Head

			for currentNode != nil && incoming.Quantity > 0 {
				restingOrder := currentNode.Value

				matchQty := incoming.Quantity
				if restingOrder.Quantity < matchQty {
					matchQty = restingOrder.Quantity
				}

				incoming.Quantity -= matchQty
				restingOrder.Quantity -= matchQty

				trades = append(trades, &TradeExecution{
					BuyOrderID:  restingOrder.ID,
					SellOrderID: incoming.ID,
					Price:       bestBidPrice,
					Quantity:    matchQty,
					Symbol:      incoming.Symbol,
				})

				if restingOrder.Quantity == 0 {
					bidLevel.Head = currentNode.Next
					if bidLevel.Head != nil {
						bidLevel.Head.Prev = nil
					} else {
						bidLevel.Tail = nil
					}
				}
				currentNode = bidLevel.Head
			}

			if bidLevel.Head == nil {
				delete(ob.Bids, bestBidPrice)
				heap.Pop(ob.BidHeap)
			}
		}

		if incoming.Quantity > 0 {
			shelf, exists := ob.Asks[incoming.Price]
			if !exists {
				shelf = &PriceLevel{Price: incoming.Price}
				ob.Asks[incoming.Price] = shelf
				heap.Push(ob.AskHeap, incoming.Price)
			}
			shelf.AppendOrder(incoming)
		}
	}

	return trades
}
// ============================================================================
// 5. THE MULTI-ASSET EXCHANGE REGISTRY (UPDATED WITH RISK & WAL)
// ============================================================================

type Exchange struct {
	Books map[[4]byte]*OrderBook
	Wal   *WAL
	Risk  *RiskEngine // Embedded Pre-Trade Risk Gatekeeper
}

func NewExchange(walInstance *WAL, riskInstance *RiskEngine) *Exchange {
	return &Exchange{
		Books: make(map[[4]byte]*OrderBook),
		Wal:   walInstance,
		Risk:  riskInstance,
	}
}

// ProcessOrder routes an order through the complete enterprise execution pipeline:
// Risk Check -> WAL Log -> Matching Engine Engine Loops.
func (e *Exchange) ProcessOrder(incoming *Order, accountID uint64) ([]*TradeExecution, error) {
	// GATE 1: Pre-Trade Risk Validation
	if e.Risk != nil {
		if err := e.Risk.EvaluateOrder(incoming, accountID); err != nil {
			return nil, err // Return the risk rejection error to the client immediately
		}
	}

	// GATE 2: Core Persistence Logging
	if e.Wal != nil {
		if err := e.Wal.WriteOrder(incoming); err != nil {
			panic("WAL CRITICAL FAILURE: Unable to persist transaction state: " + err.Error())
		}
	}

	// GATE 3: In-Memory Matching Book Execution
	book, exists := e.Books[incoming.Symbol]
	if !exists {
		book = NewOrderBook()
		e.Books[incoming.Symbol] = book
	}
	return book.ProcessOrder(incoming), nil
}
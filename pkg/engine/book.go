package engine

import (
	"container/heap"
)

// ============================================================================
// 1. DATA MODELS & STRUCTS
// ============================================================================

type Order struct {
	ID       uint64
	Symbol   [4]byte
	Price    uint64
	Quantity uint32
	IsBuy    bool
}

type OrderNode struct {
	Value *Order
	Prev  *OrderNode
	Next  *OrderNode
}

type PriceLevel struct {
	Price uint64
	Head  *OrderNode
	Tail  *OrderNode
}

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

type TradeExecution struct {
	BuyOrderID  uint64
	SellOrderID uint64
	Price       uint64
	Quantity    uint32
	Symbol      [4]byte
}

// ============================================================================
// 2. THE PRIORITY PRICE HEAP IMPLEMENTATION
// ============================================================================

// PriceHeap implements heap.Interface. It will store our active price keys.
type PriceHeap struct {
	prices []uint64
	isMin  bool // true for Asks (we want lowest price), false for Bids (we want highest price)
}

func (h *PriceHeap) Len() int { return len(h.prices) }
func (h *PriceHeap) Less(i, j int) bool {
	if h.isMin {
		return h.prices[i] < h.prices[j] // Min-Heap behavior for Sells
	}
	return h.prices[i] > h.prices[j] // Max-Heap behavior for Buys
}
func (h *PriceHeap) Swap(i, j int) { h.prices[i], h.prices[j] = h.prices[j], h.prices[i] }
func (h *PriceHeap) Push(x interface{}) {
	h.prices = append(h.prices, x.(uint64))
}
func (h *PriceHeap) Pop() interface{} {
	old := h.prices
	n := len(old)
	x := old[n-1]
	h.prices = old[0 : n-1]
	return x
}

// ============================================================================
// 3. UPGRADED HIGH-PERFORMANCE ORDER BOOK
// ============================================================================

type OrderBook struct {
	Bids      map[uint64]*PriceLevel
	Asks      map[uint64]*PriceLevel
	BidHeap   *PriceHeap // Max-Heap tracking best buy boundaries
	AskHeap   *PriceHeap // Min-Heap tracking best sell boundaries
}

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

// getBestAsk pulls the lowest sell price from the top of the Min-Heap in O(1) time
func (ob *OrderBook) getBestAsk() uint64 {
	if ob.AskHeap.Len() == 0 {
		return 0
	}
	return ob.AskHeap.prices[0]
}

// getBestBid pulls the highest buy price from the top of the Max-Heap in O(1) time
func (ob *OrderBook) getBestBid() uint64 {
	if ob.BidHeap.Len() == 0 {
		return 0
	}
	return ob.BidHeap.prices[0]
}

// ============================================================================
// 4. PRECISE MATCHING ENGINE LOOP WITH HEAP POPPING
// ============================================================================

func (ob *OrderBook) ProcessOrder(incoming *Order) []*TradeExecution {
	var trades []*TradeExecution

	if incoming.IsBuy {
		// Loop under O(1) heap evaluations
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

			// If the shelf is empty, clear the Map AND Pop the price off the Heap!
			if askLevel.Head == nil {
				delete(ob.Asks, bestAskPrice)
				heap.Pop(ob.AskHeap) // Next best sell price automatically bubbles to the top!
			}
		}

		// Queue remainder
		if incoming.Quantity > 0 {
			shelf, exists := ob.Bids[incoming.Price]
			if !exists {
				shelf = &PriceLevel{Price: incoming.Price}
				ob.Bids[incoming.Price] = shelf
				heap.Push(ob.BidHeap, incoming.Price) // Track new price level in Max-Heap
			}
			shelf.AppendOrder(incoming)
		}

	} else {
		// Incoming SELL Order
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
				bidLevel.Head = currentNode.Next
				currentNode = bidLevel.Head
			}

			if bidLevel.Head == nil {
				delete(ob.Bids, bestBidPrice)
				heap.Pop(ob.BidHeap) // Next best buy price automatically bubbles to the top!
			}
		}

		if incoming.Quantity > 0 {
			shelf, exists := ob.Asks[incoming.Price]
			if !exists {
				shelf = &PriceLevel{Price: incoming.Price}
				ob.Asks[incoming.Price] = shelf
				heap.Push(ob.AskHeap, incoming.Price) // Track new price level in Min-Heap
			}
			shelf.AppendOrder(incoming)
		}
	}

	return trades
}

// ============================================================================
// 5. THE MULTI-ASSET EXCHANGE REGISTRY
// ============================================================================

// Exchange acts as the main supervisor tracking separate books for each stock symbol.
type Exchange struct {
	Books map[[4]byte]*OrderBook // Maps a 4-byte ticker (e.g., "AAPL") to its isolated book
}

// NewExchange instantiates a fresh multi-asset supervisor.
func NewExchange() *Exchange {
	return &Exchange{
		Books: make(map[[4]byte]*OrderBook),
	}
}

// ProcessOrder routes an incoming order to its correct isolated stock book.
// If the stock book doesn't exist yet, it safely creates it on the fly.
func (e *Exchange) ProcessOrder(incoming *Order) []*TradeExecution {
	book, exists := e.Books[incoming.Symbol]
	if !exists {
		book = NewOrderBook()
		e.Books[incoming.Symbol] = book
	}

	// Delegate the order to be processed inside its specific isolated room
	return book.ProcessOrder(incoming)
}
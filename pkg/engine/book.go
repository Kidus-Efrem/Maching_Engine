package engine

import (
	"container/heap"
	"errors"
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
// 2. PRIORITY BOUNDARY PRICE HEAP
// ============================================================================

type PriceHeap struct {
	prices []uint64
	isMin  bool
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
		return h.prices[i] < h.prices[j]
	}
	return h.prices[i] > h.prices[j]
}

// ============================================================================
// 3. THE ISOLATED ORDER BOOK CONTAINER (WITH ORDERS REGISTRY)
// ============================================================================

type OrderBook struct {
	Bids           map[uint64]*PriceLevel
	Asks           map[uint64]*PriceLevel
	BidHeap        *PriceHeap
	AskHeap        *PriceHeap
	OrdersRegistry map[uint64]*OrderNode // NEW: High-speed O(1) pointer lookup tracking active orders
}

func NewOrderBook() *OrderBook {
	bidHeap := &PriceHeap{prices: make([]uint64, 0), isMin: false}
	askHeap := &PriceHeap{prices: make([]uint64, 0), isMin: true}
	heap.Init(bidHeap)
	heap.Init(askHeap)

	return &OrderBook{
		Bids:           make(map[uint64]*PriceLevel),
		Asks:           make(map[uint64]*PriceLevel),
		BidHeap:        bidHeap,
		AskHeap:        askHeap,
		OrdersRegistry: make(map[uint64]*OrderNode),
	}
}

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

func (ob *OrderBook) GetBestAsk() uint64 { return ob.getBestAsk() }
func (ob *OrderBook) GetBestBid() uint64 { return ob.getBestBid() }

// ============================================================================
// NEW: HIGH-SPEED O(1) CANCELLATION MECHANICS
// ============================================================================

func (ob *OrderBook) CancelOrder(orderID uint64) error {
	// Look up the active node pointer directly
	node, exists := ob.OrdersRegistry[orderID]
	if !exists {
		return errors.New("CANCEL_REJECT: Order is either fully filled or does not exist")
	}

	order := node.Value
	var shelf *PriceLevel
	if order.IsBuy {
		shelf = ob.Bids[order.Price]
	} else {
		shelf = ob.Asks[order.Price]
	}

	// Unlink the node from the Doubly Linked List shelf queue line
	if node.Prev != nil {
		node.Prev.Next = node.Next
	} else {
		shelf.Head = node.Next // Node was the front of the line
	}

	if node.Next != nil {
		node.Next.Prev = node.Prev
	} else {
		shelf.Tail = node.Prev // Node was the back of the line
	}

	// Clean up internal allocations
	delete(ob.OrdersRegistry, orderID)

	// If unlinking this order left the entire shelf completely empty, wipe the price layer
	if shelf.Head == nil {
		if order.IsBuy {
			delete(ob.Bids, order.Price)
			// Note: For absolute production completeness in the future, we could remove the item from the heap.
			// For now, when matching hits an empty map entry, it safely cleans up/pops automatically.
		} else {
			delete(ob.Asks, order.Price)
		}
	}

	return nil
}

// ============================================================================
// 4. PRICE-TIME ENGINE LOOP EXECUTION (WITH REGISTRY TRACKING)
// ============================================================================

func (ob *OrderBook) ProcessOrder(incoming *Order) []*TradeExecution {
	var trades []*TradeExecution

	if incoming.IsBuy {
		for incoming.Quantity > 0 && ob.AskHeap.Len() > 0 && incoming.Price >= ob.getBestAsk() {
			bestAskPrice := ob.getBestAsk()
			askLevel := ob.Asks[bestAskPrice]
			if askLevel == nil {
				heap.Pop(ob.AskHeap)
				continue
			}
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
					delete(ob.OrdersRegistry, restingOrder.ID) // Remove filled order from tracking
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
			node := shelf.AppendOrder(incoming)
			ob.OrdersRegistry[incoming.ID] = node // Track resting order location
		}

	} else {
		// Incoming Sell Order
		for incoming.Quantity > 0 && ob.BidHeap.Len() > 0 && incoming.Price <= ob.getBestBid() {
			bestBidPrice := ob.getBestBid()
			bidLevel := ob.Bids[bestBidPrice]
			if bidLevel == nil {
				heap.Pop(ob.BidHeap)
				continue
			}
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
					delete(ob.OrdersRegistry, restingOrder.ID) // Remove filled order from tracking
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
			node := shelf.AppendOrder(incoming)
			ob.OrdersRegistry[incoming.ID] = node // Track resting order location
		}
	}

	return trades
}

// ============================================================================
// 5. THE MULTI-ASSET EXCHANGE REGISTRY
// ============================================================================

type Exchange struct {
	Books map[[4]byte]*OrderBook
	Wal   *WAL
	Risk  *RiskEngine
}

func NewExchange(walInstance *WAL, riskInstance *RiskEngine) *Exchange {
	return &Exchange{
		Books: make(map[[4]byte]*OrderBook),
		Wal:   walInstance,
		Risk:  riskInstance,
	}
}

func (e *Exchange) ProcessOrder(incoming *Order, accountID uint64) ([]*TradeExecution, error) {
	if e.Risk != nil {
		if err := e.Risk.EvaluateOrder(incoming, accountID); err != nil {
			return nil, err
		}
	}

	if e.Wal != nil {
		if err := e.Wal.WriteOrder(incoming); err != nil {
			panic("WAL CRITICAL FAILURE: Unable to persist transaction state: " + err.Error())
		}
	}

	book, exists := e.Books[incoming.Symbol]
	if !exists {
		book = NewOrderBook()
		e.Books[incoming.Symbol] = book
	}
	return book.ProcessOrder(incoming), nil
}

// CancelOrder routes a cancellation request directly to the appropriate asset book
func (e *Exchange) CancelOrder(symbol [4]byte, orderID uint64) error {
	book, exists := e.Books[symbol]
	if !exists {
		return errors.New("CANCEL_REJECT: Asset ticker book does not exist")
	}
	return book.CancelOrder(orderID)
}
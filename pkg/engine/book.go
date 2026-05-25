package engine

// Order represents a user's request to buy or sell a stock.
// To keep things blazing fast and cache-friendly, we use fixed memory sizes.
type Order struct {
	ID       uint64   // Unique identifier for the order
	Symbol   [4]byte  // Fixed 4-byte array (e.g., 'A','A','P','L') instead of an expensive heap-allocated string
	Price    uint64   // Price in cents (e.g., $100.50 is stored as 10050) to prevent floating-point rounding bugs
	Quantity uint32   // Number of shares
	IsBuy    bool     // True for Buy (Bid), False for Sell (Ask)
}

// OrderNode wraps our Order struct inside a Doubly Linked List block.
type OrderNode struct {
	Value *Order     // Pointer to the actual order data
	Prev  *OrderNode // Pointer to the order ahead of this one in line
	Next  *OrderNode // Pointer to the order behind this one in line
}

// PriceLevel represents a single price "shelf" holding a queue of orders.
type PriceLevel struct {
	Price uint64     // The price value this level represents
	Head  *OrderNode // The oldest order (front of the line, matches first)
	Tail  *OrderNode // The newest order (back of the line)
}

// AppendOrder adds a new order to the very end of this price level's queue.
// This runs in O(1) constant time.
func (pl *PriceLevel) AppendOrder(order *Order) *OrderNode {
	newNode := &OrderNode{Value: order}

	if pl.Head == nil {
		// The shelf is empty; this order becomes both the front and back of the line.
		pl.Head = newNode
		pl.Tail = newNode
	} else {
		// Place the new node behind the current tail
		pl.Tail.Next = newNode
		newNode.Prev = pl.Tail
		pl.Tail = newNode // Update the tail pointer to our new node
	}
	return newNode
}

// OrderBook manages all Buy and Sell price levels for an asset.
type TradeExecution struct {
	BuyOrderID  uint64
	SellOrderID uint64
	Price       uint64
	Quantity    uint32
	Symbol      [4]byte
}

// OrderBook manages all Buy and Sell price levels for an asset.
// We added BestBid and BestAsk tracking to guarantee O(1) matching access.
type OrderBook struct {
	Bids    map[uint64]*PriceLevel // Map for O(1) direct price access
	Asks    map[uint64]*PriceLevel // Map for O(1) direct price access
	BestBid uint64                 // Highest buy price currently active
	BestAsk uint64                 // Lowest sell price currently active
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids: make(map[uint64]*PriceLevel),
		Asks: make(map[uint64]*PriceLevel),
	}
}
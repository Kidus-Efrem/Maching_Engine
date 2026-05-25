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

// ProcessOrder processes an incoming order against the book, executing trades
// immediately if a match is found, or queuing the remainder on its price shelf.
func (ob *OrderBook) ProcessOrder(incoming *Order) []*TradeExecution {
	var trades []*TradeExecution

	// Case A: Incoming Order is a BUY order
	if incoming.IsBuy {
		// Keep matching as long as there are shares left AND there is a seller willing to meet the price
		for incoming.Quantity > 0 && ob.BestAsk != 0 && incoming.Price >= ob.BestAsk {
			askLevel := ob.Asks[ob.BestAsk]
			currentNode := askLevel.Head

			// Walk down the linked list queue at this specific price shelf
			for currentNode != nil && incoming.Quantity > 0 {
				restingOrder := currentNode.Value

				// Determine execution quantity (the maximum available match amount)
				matchQty := incoming.Quantity
				if restingOrder.Quantity < matchQty {
					matchQty = restingOrder.Quantity
				}

				// Deduct stock balances
				incoming.Quantity -= matchQty
				restingOrder.Quantity -= matchQty

				// Generate the trade receipt
				trades = append(trades, &TradeExecution{
					BuyOrderID:  incoming.ID,
					SellOrderID: restingOrder.ID,
					Price:       ob.BestAsk, // Price matches the existing resting order
					Quantity:    matchQty,
					Symbol:      incoming.Symbol,
				})

				// If the resting order is entirely filled, pop it out of our linked list
				if restingOrder.Quantity == 0 {
					askLevel.Head = currentNode.Next
					if askLevel.Head != nil {
						askLevel.Head.Prev = nil
					} else {
						askLevel.Tail = nil // The shelf is completely empty of orders
					}
				}
				currentNode = askLevel.Head
			}

			// If the entire price shelf was cleared out, remove it from our map and find the next best price
			if askLevel.Head == nil {
				delete(ob.Asks, ob.BestAsk)
				ob.recalculateBestAsk()
			}
		}

		// If the incoming buy order still has remaining shares left after matching, shelf it
		if incoming.Quantity > 0 {
			shelf, exists := ob.Bids[incoming.Price]
			if !exists {
				shelf = &PriceLevel{Price: incoming.Price}
				ob.Bids[incoming.Price] = shelf
			}
			shelf.AppendOrder(incoming)
			if incoming.Price > ob.BestBid {
				ob.BestBid = incoming.Price
			}
		}

	} else {
		// Case B: Incoming Order is a SELL order (Mirrors the Buy logic perfectly)
		for incoming.Quantity > 0 && ob.BestBid != 0 && incoming.Price <= ob.BestBid {
			bidLevel := ob.Bids[ob.BestBid]
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
					Price:       ob.BestBid,
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
				delete(ob.Bids, ob.BestBid)
				ob.recalculateBestBid()
			}
		}

		if incoming.Quantity > 0 {
			shelf, exists := ob.Asks[incoming.Price]
			if !exists {
				shelf = &PriceLevel{Price: incoming.Price}
				ob.Asks[incoming.Price] = shelf
			}
			shelf.AppendOrder(incoming)
			if ob.BestAsk == 0 || incoming.Price < ob.BestAsk {
				ob.BestAsk = incoming.Price
			}
		}
	}

	return trades
}

// Helper methods to dynamically scan boundaries when a price shelf is completely wiped out.
func (ob *OrderBook) recalculateBestBid() {
	var max uint64
	for price := range ob.Bids {
		if price > max {
			max = price
		}
	}
	ob.BestBid = max
}

func (ob *OrderBook) recalculateBestAsk() {
	var min uint64 = 0
	for price := range ob.Asks {
		if min == 0 || price < min {
			min = price
		}
	}
	ob.BestAsk = min
}
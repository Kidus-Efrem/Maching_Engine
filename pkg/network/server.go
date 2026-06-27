package network

import (
	"encoding/binary"
	"errors"
	"matching-engine/pkg/engine"
)

// PacketSize defines our strict 33-byte protocol layout boundary
const PacketSize = 34 // 1 + 8 + 8 + 4 + 8 + 4 + 1

// UnmarshalOrder decodes a raw 34-byte slice directly into an Engine Order struct
// without causing any dynamic memory allocations on the heap.
func UnmarshalOrder(data []byte) (engine.Order, uint64, error) {
	if len(data) < PacketSize {
		return engine.Order{}, 0, errors.New("malformed packet: insufficient byte length")
	}

	// Offset 0: MsgType (e.g., 'O' for Order)
	msgType := data[0]
	if msgType != 'O' {
		return engine.Order{}, 0, errors.New("unsupported message type")
	}

	// Offset 1-8: OrderID (uint64)
	orderID := binary.BigEndian.Uint64(data[1:9])

	// Offset 9-16: AccountID (uint64)
	accountID := binary.BigEndian.Uint64(data[9:17])

	// Offset 17-20: Symbol ([4]byte)
	var symbol [4]byte
	copy(symbol[:], data[17:21])

	// Offset 21-28: Price (uint64)
	price := binary.BigEndian.Uint64(data[21:29])

	// Offset 29-32: Quantity (uint32)
	quantity := binary.BigEndian.Uint32(data[29:33])

	// Offset 33: IsBuy (bool represented as a single byte 1 or 0)
	isBuy := data[33] == 1

	// Construct our stack-allocated value order footprint
	order := engine.Order{
		ID:       orderID,
		Symbol:   symbol,
		Price:    price,
		Quantity: quantity,
		IsBuy:    isBuy,
	}

	return order, accountID, nil
}
package engine

import (
	"encoding/binary"
	"io"
	"os"
)

type SnapshotEngine struct {
	SnapshotPath string
}

func NewSnapshotEngine(path string) *SnapshotEngine {
	return &SnapshotEngine{SnapshotPath: path}
}

// SaveExchangeSnapshot freezes the global multi-ticker exchange state to disk
func (se *SnapshotEngine) SaveExchangeSnapshot(exchange *Exchange) error {
	file, err := os.OpenFile(se.SnapshotPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	// 1. Write how many unique asset ticker books currently exist in the exchange
	bookCount := uint32(len(exchange.Books))
	if err := binary.Write(file, binary.BigEndian, bookCount); err != nil {
		return err
	}

	// 2. Loop through every asset ticker book (e.g., AAPL, MSFT)
	for symbol, book := range exchange.Books {
		// Write the 4-byte ticker key frame first so we know which book this belongs to
		if _, err := file.Write(symbol[:]); err != nil {
			return err
		}

		// Write how many active orders live inside this specific ticker's registry
		registrySize := uint64(len(book.OrdersRegistry))
		if err := binary.Write(file, binary.BigEndian, registrySize); err != nil {
			return err
		}

		// Dump every single order belonging to this symbol block
		var buf [21]byte
		for _, node := range book.OrdersRegistry {
			order := node.Value
			binary.BigEndian.PutUint64(buf[0:8], order.ID)
			copy(buf[8:12], order.Symbol[:])
			binary.BigEndian.PutUint64(buf[12:20], order.Price)
			if order.IsBuy {
				buf[20] = 1
			} else {
				buf[20] = 0
			}

			if _, err := file.Write(buf[:]); err != nil {
				return err
			}
			if err := binary.Write(file, binary.BigEndian, order.Quantity); err != nil {
				return err
			}
		}
	}

	return nil
}

// LoadExchangeSnapshot completely rehydrates a full multi-asset Exchange platform
func (se *SnapshotEngine) LoadExchangeSnapshot() (*Exchange, error) {
	file, err := os.Open(se.SnapshotPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a fresh, pristine global exchange registry container
	exchange := NewExchange(nil, nil)

	// 1. Read how many separate asset ticker books we need to parse out of the stream
	var bookCount uint32
	if err := binary.Read(file, binary.BigEndian, &bookCount); err != nil {
		return nil, err
	}

	// 2. Loop through each asset block segment
	buf := make([]byte, 21)
	for b := uint32(0); b < bookCount; b++ {
		var symbol [4]byte
		if _, err := io.ReadFull(file, symbol[:]); err != nil {
			return nil, err
		}

		// Initialize an isolated book room for this specific symbol inside the registry
		book := NewOrderBook()
		exchange.Books[symbol] = book

		// Read how many orders are packed inside this symbol room block
		var registrySize uint64
		if err := binary.Read(file, binary.BigEndian, &registrySize); err != nil {
			return nil, err
		}

		// Parse and inject each order back to its isolated asset space
		for i := uint64(0); i < registrySize; i++ {
			_, err := io.ReadFull(file, buf)
			if err != nil {
				return nil, err
			}

			order := &Order{
				ID:    binary.BigEndian.Uint64(buf[0:8]),
				Price: binary.BigEndian.Uint64(buf[12:20]),
				IsBuy: buf[20] == 1,
			}
			copy(order.Symbol[:], buf[8:12])

			var quantity uint32
			if err := binary.Read(file, binary.BigEndian, &quantity); err != nil {
				return nil, err
			}
			order.Quantity = quantity

			// Process this order *directly* inside its specific symbol room
			book.ProcessOrder(order)
		}
	}

	return exchange, nil
}
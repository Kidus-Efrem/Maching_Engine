package engine

import (
	"encoding/binary"
	"io"
	"os"
)

// SnapshotEngine handles freezing and rehydrating the OrderBook states.
type SnapshotEngine struct {
	SnapshotPath string
}

func NewSnapshotEngine(path string) *SnapshotEngine {
	return &SnapshotEngine{SnapshotPath: path}
}

// SaveSnapshot serializes the current OrderBook data structure directly to disk.
func (se *SnapshotEngine) SaveSnapshot(book *OrderBook) error {
	file, err := os.OpenFile(se.SnapshotPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	// 1. Write the total number of tracking entries inside our OrdersRegistry
	registrySize := uint64(len(book.OrdersRegistry))
	if err := binary.Write(file, binary.BigEndian, registrySize); err != nil {
		return err
	}

	// 2. Iterate through the registry map and dump every order's raw fields
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

		// Also serialize the current remaining quantity of the order
		// We write this separately so we know exactly how much volume was left unfilled!
		if _, err := file.Write(buf[:]); err != nil {
			return err
		}
		if err := binary.Write(file, binary.BigEndian, order.Quantity); err != nil {
			return err
		}
	}

	return nil
}

// LoadSnapshot reads the snapshot file and rehydrates a pristine OrderBook memory layer.
func (se *SnapshotEngine) LoadSnapshot() (*OrderBook, error) {
	file, err := os.Open(se.SnapshotPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	book := NewOrderBook()

	// 1. Read the total number of records we need to parse
	var registrySize uint64
	if err := binary.Read(file, binary.BigEndian, &registrySize); err != nil {
		return nil, err
	}

	// 2. Rehydrate each individual order entry back into our Heaps and Maps
	buf := make([]byte, 21)
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

		// Force place this order back into our live map/heap queues
		book.ProcessOrder(order)
	}

	return book, nil
}

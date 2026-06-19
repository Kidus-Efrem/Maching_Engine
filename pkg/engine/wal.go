package engine

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// Const defining our fixed binary record footprint (8 + 4 + 8 + 1 + 4 = 25 bytes)
const OrderRecordSize = 25

// WAL manages our low-level append-only file descriptor interface.
type WAL struct {
	file *os.File
}

// OpenWAL initializes or restores an append-only binary log file on disk.
func OpenWAL(path string) (*WAL, error) {
	// Open file with Read/Write, Create if missing, and Append-Only flags.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &WAL{file: file}, nil
}

// Close gracefully releases the file descriptor back to the OS.
func (w *WAL) Close() error {
	return w.file.Close()
}

// WriteOrder serializes an order structure directly into a tight 25-byte block.
func (w *WAL) WriteOrder(order *Order) error {
	var buf [OrderRecordSize]byte

	// Bit-pack the fields sequentially into the byte array buffer
	binary.BigEndian.PutUint64(buf[0:8], order.ID)
	copy(buf[8:12], order.Symbol[:])
	binary.BigEndian.PutUint64(buf[12:20], order.Price)

	// Encode the boolean field into bit 0 of the 21st byte frame
	if order.IsBuy {
		buf[20] = 1
	} else {
		buf[20] = 0
	}

	// Pack the 4-byte Quantity into the final slots
	binary.BigEndian.PutUint32(buf[21:25], order.Quantity)

	// Direct append write system call
	_, err := w.file.Write(buf[:])
	return err
}

// ReadAll iterates through the log and parses raw chunks back into memory state arrays.
func (w *WAL) ReadAll() ([]*Order, error) {
	// Seek back to the absolute beginning of our file descriptor
	_, err := w.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	var orders []*Order
	buf := make([]byte, OrderRecordSize)

	for {
		_, err := io.ReadFull(w.file, buf)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break // Safe file termination boundary reached
			}
			return nil, err
		}

		// Rehydrate our Order struct out of raw binary streams
		order := &Order{
			ID:       binary.BigEndian.Uint64(buf[0:8]),
			Price:    binary.BigEndian.Uint64(buf[12:20]),
			IsBuy:    buf[20] == 1,
			Quantity: binary.BigEndian.Uint32(buf[21:25]), // Fixed: Rehydrating volume
		}
		copy(order.Symbol[:], buf[8:12])

		orders = append(orders, order)
	}

	return orders, nil
}
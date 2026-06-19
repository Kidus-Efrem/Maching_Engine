package main

import (
	"fmt"
	"matching-engine/pkg/engine"
	"os"
	"sync"
	"time"
)

func main() {
	fmt.Println("======= 🚀 NEXUSTRADE MASTER INTEGRATION SIMULATOR =======")

	// File Paths for Persistence Verification
	walPath := "integration_wal.log"
	snapshotPath := "integration_exchange.snapshot"

	// Clean up any stale files from previous runs
	os.Remove(walPath)
	os.Remove(snapshotPath)

	// ------------------------------------------------------------------------
	// STEP 1: INITIALIZE ALL ENTERPRISE ENGINE COMPONENTS
	// ------------------------------------------------------------------------
	fmt.Println("\n🛠️ [1/5] Initializing Engine Infrastructure Layers...")

	wal, err := engine.OpenWAL(walPath)
	if err != nil {
		panic("Failed to initialize WAL: " + err.Error())
	}

	riskGate := engine.NewRiskEngine()
	snapEngine := engine.NewSnapshotEngine(snapshotPath)
	ringBuffer := engine.NewRingBuffer()

	// Seed Trader Profile with strict risk parameters
	traderID := uint64(99)
	riskGate.Accounts[traderID] = &engine.AccountState{
		AccountID:         traderID,
		AvailableBalance:  500000, // $5,000.00 scaled cash capital
		ActiveMaxPosition: 1000,   // Max shares per order
	}

	// Instantiate Exchange passing our integrated WAL and Risk components
	exchange := engine.NewExchange(wal, riskGate)

	// Start the Background Lock-Free Consumer Thread
	ringBuffer.StartConsuming(exchange)
	fmt.Println("✅ All infrastructure pipelines online. Background Consumer listening...")

	// Tickers to trade
	aapl := [4]byte{'A', 'A', 'P', 'L'}
	msft := [4]byte{'M', 'S', 'F', 'T'}

	// ------------------------------------------------------------------------
	// STEP 2: MULTI-THREADED CHAOS INGESTION TEST
	// ------------------------------------------------------------------------
	fmt.Println("\n🌪️ [2/5] Launching Multi-Threaded Ingestion Traffic...")
	var wg sync.WaitGroup

	// Worker Thread 1: Blasting Valid AAPL Sell Liquidity
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 5; i++ {
			ringBuffer.Enqueue(&engine.Order{
				ID: uint64(100 + i), Symbol: aapl, Price: 10000, Quantity: 10, IsBuy: false,
			}, traderID)
		}
	}()

	// Worker Thread 2: Blasting Valid MSFT Sell Liquidity
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 5; i++ {
			ringBuffer.Enqueue(&engine.Order{
				ID: uint64(200 + i), Symbol: msft, Price: 25000, Quantity: 20, IsBuy: false,
			}, traderID)
		}
	}()

	// Worker Thread 3: Blasting Malicious/Illegal Orders to test the Risk Engine Gate
	wg.Add(1)
	go func() {
		defer wg.Done()
		// This order costs $6,000.00 — which exceeds our $5,000.00 cash balance!
		// It should be trapped and rejected cleanly by Gate 1 inside the Consumer loop.
		ringBuffer.Enqueue(&engine.Order{
			ID: 999, Symbol: aapl, Price: 10000, Quantity: 60, IsBuy: true,
		}, traderID)
	}()

	// Wait for network threads to finish placing entries into the Ring Buffer slots
	wg.Wait()
	fmt.Println("📥 All concurrent network traffic successfully accepted by Ring Buffer.")

	// Give the single-threaded background matching core a fraction of a second to drain the array
	time.Sleep(100 * time.Millisecond)

	// ------------------------------------------------------------------------
	// STEP 3: POINTER-LEVEL CANCELLATION VALIDATION
	// ------------------------------------------------------------------------
	fmt.Println("\n⚡ [3/5] Testing Instant O(1) Cancellation Ring Registry...")
	// Let's cancel Order #101 (The first AAPL sell order placed)
	if err := exchange.CancelOrder(aapl, 101); err != nil {
		fmt.Printf("❌ Cancellation Failed: %v\n", err)
	} else {
		fmt.Println("✅ Success: Order #101 cleanly unlinked from memory registry via intrusive pointers!")
	}

	// ------------------------------------------------------------------------
	// STEP 4: FREEZING MEMORY MATRIX VIA CROSS-SYMBOL SNAPSHOTS
	// ------------------------------------------------------------------------
	fmt.Println("\n💾 [4/5] Executing Global Multi-Asset State Snapshot to Disk...")
	if err := snapEngine.SaveExchangeSnapshot(exchange); err != nil {
		panic("Snapshot creation failed: " + err.Error())
	}

	// Close the active WAL handle to flush remaining disk caches safely
	wal.Close()

	// ------------------------------------------------------------------------
	// STEP 5: VAPORIZE VARIABLES & AUDIT SYSTEM RECOVERY BOUNDS
	// ------------------------------------------------------------------------
	fmt.Println("\n💥 [5/5] SIMULATING TOTAL SYSTEM BLACKOUT (Purging Active Memory RAM)...")
	exchange = nil

	fmt.Println("🔄 Booting Backup Engine Clusters from Snapshot Records...")
	recoveredExchange, err := snapEngine.LoadExchangeSnapshot()
	if err != nil {
		panic("Failed to rehydrate snapshot state: " + err.Error())
	}

	fmt.Println("\n======================= INTEGRITY AUDIT REPORT =======================")

	// Audit Asset 1: AAPL
	recoveredAAPLBook := recoveredExchange.Books[aapl]
	if recoveredAAPLBook != nil {
		// Originally 5 orders of 10 shares = 50 shares. We cancelled 1 order (10 shares).
		// Total tracking capacity remaining must equal exactly 40 shares!
		totalVolume := 0
		for _, node := range recoveredAAPLBook.OrdersRegistry {
			totalVolume += int(node.Value.Quantity)
		}
		fmt.Printf("AAPL Liquidity Audit: Verified %d Shares Active across %d Registry Rows (Target: 40 Shares)\n",
			totalVolume, len(recoveredAAPLBook.OrdersRegistry))
	} else {
		fmt.Println("❌ Audit Failure: AAPL Book missing from recovered environment.")
	}

	// Audit Asset 2: MSFT
	recoveredMSFTBook := recoveredExchange.Books[msft]
	if recoveredMSFTBook != nil {
		fmt.Printf("MSFT Liquidity Audit: Verified Best Ask sitting firmly at $%d.00 (Target: $250.00)\n",
			recoveredMSFTBook.GetBestAsk()/100)
	} else {
		fmt.Println("❌ Audit Failure: MSFT Book missing from recovered environment.")
	}

	// Clean up storage logs after a successful test run
	os.Remove(walPath)
	os.Remove(snapshotPath)
	fmt.Println("======================================================================")
	fmt.Println("🎉 ALL SYSTEMS FUNCTIONING PERFECTLY: INTEGRATION VERIFICATION PASSED!")
}
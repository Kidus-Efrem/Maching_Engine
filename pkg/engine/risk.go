package engine

import "errors"

// AccountState tracks the financial risk profiles for individual traders.
type AccountState struct {
	AccountID         uint64
	AvailableBalance  uint64 // Scaled integer cash balance (e.g., $10,000.00 is 1000000)
	ActiveMaxPosition uint32 // Maximum allowed shares per single order transaction
}

// RiskEngine acts as the pre-trade validation gatekeeper for the exchange.
type RiskEngine struct {
	Accounts map[uint64]*AccountState
}

// NewRiskEngine allocates memory for our pre-trade risk tracking system.
func NewRiskEngine() *RiskEngine {
	return &RiskEngine{
		Accounts: make(map[uint64]*AccountState),
	}
}

// EvaluateOrder inspects an incoming order against the trader's credit parameters.
// This operation runs in O(1) time complexity to maintain low-latency guarantees.
func (re *RiskEngine) EvaluateOrder(order *Order, accountID uint64) error {
	account, exists := re.Accounts[accountID]
	if !exists {
		return errors.New("RISK_REJECT: Unknown trader account profile identifier")
	}

	// Rule 1: Check for fat-finger quantity limit violations
	if order.Quantity > account.ActiveMaxPosition {
		return errors.New("RISK_REJECT: Order volume exceeds maximum allowed position limit")
	}

	// Rule 2: If buying, verify the account has sufficient available capital
	if order.IsBuy {
		totalCost := uint64(order.Quantity) * order.Price
		if totalCost > account.AvailableBalance {
			return errors.New("RISK_REJECT: Insufficient account credit balance for transaction")
		}
	}

	return nil // Order passed all pre-trade risk criteria successfully
}

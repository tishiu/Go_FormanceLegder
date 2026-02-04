package ledger

import (
	"fmt"
	"math/big"
)

func validateDoubleEntry(cmd PostTransactionCommand, accounts map[string]Account) error {
	if len(cmd.Postings) < 2 {
		return fmt.Errorf("transaction must have at least 2 postings")
	}

	// Group by currency and sum debits/credits
	totalDebits := new(big.Rat)
	totalCredits := new(big.Rat)

	for _, p := range cmd.Postings {
		// Verify account exists
		if _, ok := accounts[p.AccountCode]; !ok {
			return fmt.Errorf("account %s not found", p.AccountCode)
		}

		// Verify direction
		if p.Direction != "debit" && p.Direction != "credit" {
			return fmt.Errorf("invalid direction: %s", p.Direction)
		}

		// Parse amount
		amount := new(big.Rat)
		if _, ok := amount.SetString(p.Amount); !ok {
			return fmt.Errorf("invalid amount: %s", p.Amount)
		}

		// Check positive
		if amount.Sign() <= 0 {
			return fmt.Errorf("amount must be positive: %s", p.Amount)
		}

		// Accumulate
		if p.Direction == "debit" {
			totalDebits.Add(totalDebits, amount)
		} else {
			totalCredits.Add(totalCredits, amount)
		}
	}

	// Verify balance
	if totalDebits.Cmp(totalCredits) != 0 {
		return fmt.Errorf("debits (%s) must equal credits (%s)", totalDebits.FloatString(10), totalCredits.FloatString(10))
	}

	return nil
}

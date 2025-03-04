package internal

// Balance represents an account balance
type Balance struct {
	Currency string
	Amount   float64
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
)

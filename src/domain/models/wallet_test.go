package models

import (
	"testing"
	"time"
)

func TestNewWallet(t *testing.T) {
	name := "Test Wallet"
	description := "A wallet for testing"
	currency := "USD"
	walletType := WalletTypeBank

	wallet := NewWallet(name, description, currency, walletType)

	if wallet.ID == "" {
		t.Error("Expected new wallet to have an ID, but it was empty")
	}
	if wallet.Name != name {
		t.Errorf("Expected wallet name to be '%s', but got '%s'", name, wallet.Name)
	}
	if wallet.Description != description {
		t.Errorf("Expected wallet description to be '%s', but got '%s'", description, wallet.Description)
	}
	if wallet.Currency != currency {
		t.Errorf("Expected wallet currency to be '%s', but got '%s'", currency, wallet.Currency)
	}
	if wallet.Type != walletType {
		t.Errorf("Expected wallet type to be '%s', but got '%s'", walletType, wallet.Type)
	}
	if wallet.Balance != 0 {
		t.Errorf("Expected initial balance to be 0, but got %f", wallet.Balance)
	}
	if wallet.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set, but it was zero")
	}
	if wallet.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set, but it was zero")
	}
}

func TestWallet_Validate(t *testing.T) {
	tests := []struct {
		name      string
		wallet    *Wallet
		expectErr bool
		errType   error
	}{
		{
			name: "Valid Wallet",
			wallet: &Wallet{
				ID:       "wallet-1",
				Name:     "Checking Account",
				Currency: "USD",
				Type:     WalletTypeBank,
			},
			expectErr: false,
		},
		{
			name: "Missing Name",
			wallet: &Wallet{
				ID:       "wallet-2",
				Name:     "", // Missing name
				Currency: "EUR",
				Type:     WalletTypeBank,
			},
			expectErr: true,
			errType:   ErrMissingWalletName,
		},
		{
			name: "Missing Currency",
			wallet: &Wallet{
				ID:       "wallet-3",
				Name:     "Bitcoin Wallet",
				Currency: "", // Missing currency
				Type:     WalletTypeCrypto,
			},
			expectErr: true,
			errType:   ErrMissingCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wallet.Validate()
			hasErr := err != nil

			if hasErr != tt.expectErr {
				t.Errorf("Validate() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if tt.expectErr && err != tt.errType {
				t.Errorf("Validate() error type = %T, want %T (%v)", err, tt.errType, tt.errType)
			}
		})
	}
}

func TestWallet_UpdateBalance(t *testing.T) {
	wallet := NewWallet("Test", "", "USD", WalletTypeBank)
	initialBalance := wallet.Balance
	initialUpdatedAt := wallet.UpdatedAt

	time.Sleep(1 * time.Millisecond) // Ensure UpdatedAt changes

	amount := 100.50
	wallet.UpdateBalance(amount)

	expectedBalance := initialBalance + amount
	if wallet.Balance != expectedBalance {
		t.Errorf("Expected balance to be %f, but got %f", expectedBalance, wallet.Balance)
	}

	if !wallet.UpdatedAt.After(initialUpdatedAt) {
		t.Errorf("Expected UpdatedAt (%v) to be after initial UpdatedAt (%v)", wallet.UpdatedAt, initialUpdatedAt)
	}

	amount = -50.25
	wallet.UpdateBalance(amount)
	expectedBalance += amount
	if wallet.Balance != expectedBalance {
		t.Errorf("Expected balance to be %f after decrease, but got %f", expectedBalance, wallet.Balance)
	}
}

func TestWallet_HasSufficientBalance(t *testing.T) {
	wallet := NewWallet("Test", "", "USD", WalletTypeBank)
	wallet.Balance = 100.0

	tests := []struct {
		name   string
		amount float64
		want   bool
	}{
		{"Sufficient", 50.0, true},
		{"Exact", 100.0, true},
		{"Insufficient", 100.01, false},
		{"Zero Amount", 0.0, true},
		{"Negative Amount", -10.0, true}, // Technically sufficient, though withdrawal logic might prevent negative amounts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wallet.HasSufficientBalance(tt.amount); got != tt.want {
				t.Errorf("HasSufficientBalance(%f) = %v, want %v", tt.amount, got, tt.want)
			}
		})
	}
}

func TestWallet_ProcessIncome(t *testing.T) {
	wallet := NewWallet("Test", "", "USD", WalletTypeBank)
	initialBalance := 50.0
	wallet.Balance = initialBalance
	amount := 100.0

	wallet.ProcessIncome(amount)

	expectedBalance := initialBalance + amount
	if wallet.Balance != expectedBalance {
		t.Errorf("Expected balance after income to be %f, but got %f", expectedBalance, wallet.Balance)
	}
}

func TestWallet_ProcessExpense(t *testing.T) {
	t.Run("Sufficient Funds", func(t *testing.T) {
		wallet := NewWallet("Test", "", "USD", WalletTypeBank)
		initialBalance := 100.0
		wallet.Balance = initialBalance
		amount := 75.0

		err := wallet.ProcessExpense(amount)
		if err != nil {
			t.Errorf("ProcessExpense() returned unexpected error: %v", err)
		}

		expectedBalance := initialBalance - amount
		if wallet.Balance != expectedBalance {
			t.Errorf("Expected balance after expense to be %f, but got %f", expectedBalance, wallet.Balance)
		}
	})

	t.Run("Insufficient Funds", func(t *testing.T) {
		wallet := NewWallet("Test", "", "USD", WalletTypeBank)
		initialBalance := 50.0
		wallet.Balance = initialBalance
		amount := 75.0

		err := wallet.ProcessExpense(amount)
		if err == nil {
			t.Error("ProcessExpense() expected an error for insufficient funds, but got nil")
		} else if err != ErrInsufficientBalance {
			t.Errorf("ProcessExpense() expected error type %T, but got %T (%v)", ErrInsufficientBalance, err, err)
		}

		// Balance should not change
		if wallet.Balance != initialBalance {
			t.Errorf("Expected balance to remain %f after failed expense, but got %f", initialBalance, wallet.Balance)
		}
	})
}

func TestWallet_ProcessTransferOut(t *testing.T) {
	// ProcessTransferOut uses ProcessExpense, so we just need a basic check
	t.Run("Sufficient Funds", func(t *testing.T) {
		wallet := NewWallet("Test", "", "USD", WalletTypeBank)
		initialBalance := 100.0
		wallet.Balance = initialBalance
		amount := 75.0

		err := wallet.ProcessTransferOut(amount)
		if err != nil {
			t.Errorf("ProcessTransferOut() returned unexpected error: %v", err)
		}
		expectedBalance := initialBalance - amount
		if wallet.Balance != expectedBalance {
			t.Errorf("Expected balance after transfer out to be %f, but got %f", expectedBalance, wallet.Balance)
		}
	})

	t.Run("Insufficient Funds", func(t *testing.T) {
		wallet := NewWallet("Test", "", "USD", WalletTypeBank)
		initialBalance := 50.0
		wallet.Balance = initialBalance
		amount := 75.0

		err := wallet.ProcessTransferOut(amount)
		if err == nil {
			t.Error("ProcessTransferOut() expected an error for insufficient funds, but got nil")
		} else if err != ErrInsufficientBalance {
			t.Errorf("ProcessTransferOut() expected error type %T, but got %T (%v)", ErrInsufficientBalance, err, err)
		}
		if wallet.Balance != initialBalance {
			t.Errorf("Expected balance to remain %f after failed transfer out, but got %f", initialBalance, wallet.Balance)
		}
	})
}

func TestWallet_ProcessTransferIn(t *testing.T) {
	t.Run("No Exchange Rate", func(t *testing.T) {
		wallet := NewWallet("Test", "", "USD", WalletTypeBank)
		initialBalance := 50.0
		wallet.Balance = initialBalance
		amount := 100.0
		exchangeRate := 0.0 // No rate

		wallet.ProcessTransferIn(amount, exchangeRate)

		expectedBalance := initialBalance + amount
		if wallet.Balance != expectedBalance {
			t.Errorf("Expected balance after transfer in (no rate) to be %f, but got %f", expectedBalance, wallet.Balance)
		}
	})

	t.Run("With Exchange Rate", func(t *testing.T) {
		wallet := NewWallet("Test", "", "EUR", WalletTypeBank)
		initialBalance := 50.0
		wallet.Balance = initialBalance
		amount := 100.0     // Amount in source currency (e.g., USD)
		exchangeRate := 0.9 // 1 USD = 0.9 EUR

		wallet.ProcessTransferIn(amount, exchangeRate)

		expectedAmountInWalletCurrency := amount * exchangeRate
		expectedBalance := initialBalance + expectedAmountInWalletCurrency
		if wallet.Balance != expectedBalance {
			t.Errorf("Expected balance after transfer in (with rate) to be %f, but got %f", expectedBalance, wallet.Balance)
		}
	})

	t.Run("Negative Exchange Rate", func(t *testing.T) {
		wallet := NewWallet("Test", "", "USD", WalletTypeBank)
		initialBalance := 50.0
		wallet.Balance = initialBalance
		amount := 100.0
		exchangeRate := -0.9 // Invalid rate, should be treated as 0

		wallet.ProcessTransferIn(amount, exchangeRate)

		expectedBalance := initialBalance + amount // Should ignore negative rate
		if wallet.Balance != expectedBalance {
			t.Errorf("Expected balance after transfer in (negative rate) to be %f, but got %f", expectedBalance, wallet.Balance)
		}
	})
}

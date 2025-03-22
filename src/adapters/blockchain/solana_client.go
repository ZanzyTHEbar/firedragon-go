package blockchain

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/firefly"
)

// SolanaClient implements the BlockchainClient interface for Solana
type SolanaClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewSolanaClient creates a new Solana client
func NewSolanaClient() *SolanaClient {
	return &SolanaClient{
		endpoint: "https://api.solscan.io",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchTransactions retrieves transactions for a Solana address
func (c *SolanaClient) FetchTransactions(address string) ([]firefly.TransactionModel, error) {
	url := fmt.Sprintf("%s/account/transactions?account=%s&limit=50", c.endpoint, address)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch solana transactions: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool
		Data    []struct {
			TxHash    string
			Timestamp int64
			Status    string
			Fee       float64
			FromAddr  string
			ToAddr    string
			Amount    float64
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode solana response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("solana API error")
	}

	var transactions []firefly.TransactionModel
	for _, tx := range result.Data {
		// Skip failed transactions
		if tx.Status != "Success" {
			continue
		}

		// Convert timestamp to time.Time
		timestamp := time.Unix(tx.Timestamp, 0)

		// Determine transaction type
		transType := "withdrawal"
		if tx.ToAddr == address {
			transType = "deposit"
		}

		transactions = append(transactions, firefly.TransactionModel{
			ID:          tx.TxHash,
			Currency:    "SOL",
			Amount:      tx.Amount,
			TransType:   transType,
			Description: fmt.Sprintf("Solana transaction %s", tx.TxHash),
			Date:        timestamp,
		})
	}

	return transactions, nil
}

// GetBalance retrieves the current balance for a Solana address
func (c *SolanaClient) GetBalance(address string) (float64, error) {
	url := fmt.Sprintf("%s/account/tokens?account=%s", c.endpoint, address)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch solana balance: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool
		Data    []struct {
			TokenAddress string
			Balance      float64
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode solana balance response: %w", err)
	}

	if !result.Success {
		return 0, fmt.Errorf("solana API error")
	}

	// Find SOL balance (native token)
	for _, token := range result.Data {
		if token.TokenAddress == "SOL" {
			return token.Balance, nil
		}
	}

	return 0, nil
}

// GetName returns the name of the blockchain
func (c *SolanaClient) GetName() string {
	return "solana"
}

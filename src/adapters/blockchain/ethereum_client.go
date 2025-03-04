package blockchain

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/firefly"
)

// EthereumClient implements the BlockchainClient interface for Ethereum
type EthereumClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewEthereumClient creates a new Ethereum client
func NewEthereumClient(apiKey string) *EthereumClient {
	return &EthereumClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchTransactions retrieves transactions for an Ethereum address
func (c *EthereumClient) FetchTransactions(address string) ([]firefly.CustomTransaction, error) {
	url := fmt.Sprintf("https://api.etherscan.io/api?module=account&action=txlist&address=%s&sort=desc&apikey=%s",
		address, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ethereum transactions: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Status  string
		Message string
		Result  []struct {
			Hash        string
			From        string
			To          string
			Value       string
			TimeStamp   string
			IsError     string
			BlockNumber string
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode ethereum response: %w", err)
	}

	if result.Status != "1" {
		return nil, fmt.Errorf("ethereum API error: %s", result.Message)
	}

	var transactions []firefly.CustomTransaction
	for _, tx := range result.Result {
		// Skip failed transactions
		if tx.IsError == "1" {
			continue
		}

		// Convert timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05", tx.TimeStamp)
		if err != nil {
			continue
		}

		// Determine transaction type
		transType := "withdrawal"
		if tx.To == address {
			transType = "deposit"
		}

		transactions = append(transactions, firefly.CustomTransaction{
			ID:          tx.Hash,
			Currency:    "ETH",
			Amount:      0.0, // TODO: Convert Value from Wei to ETH
			TransType:   transType,
			Description: fmt.Sprintf("Ethereum transaction %s", tx.Hash),
			Date:        timestamp,
		})
	}

	return transactions, nil
}

// GetBalance retrieves the current balance for an Ethereum address
func (c *EthereumClient) GetBalance(address string) (float64, error) {
	url := fmt.Sprintf("https://api.etherscan.io/api?module=account&action=balance&address=%s&tag=latest&apikey=%s",
		address, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch ethereum balance: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Status  string
		Message string
		Result  string
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode ethereum balance response: %w", err)
	}

	if result.Status != "1" {
		return 0, fmt.Errorf("ethereum API error: %s", result.Message)
	}

	// TODO: Convert balance from Wei to ETH
	return 0.0, nil
}

// GetName returns the name of the blockchain
func (c *EthereumClient) GetName() string {
	return "ethereum"
}

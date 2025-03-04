package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/firefly"
)

// SUIClient implements the BlockchainClient interface for SUI
type SUIClient struct {
	rpcURL     string
	httpClient *http.Client
}

// NewSUIClient creates a new SUI client
func NewSUIClient() *SUIClient {
	return &SUIClient{
		rpcURL: "https://fullnode.mainnet.sui.io:443",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRPCCall makes a JSON-RPC call to the SUI node
func (c *SUIClient) makeRPCCall(method string, params []interface{}) (json.RawMessage, error) {
	request := struct {
		JsonRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}{
		JsonRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.rpcURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to make RPC call: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Error  *struct{ Message string }
		Result json.RawMessage
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", result.Error.Message)
	}

	return result.Result, nil
}

// FetchTransactions retrieves transactions for a SUI address
func (c *SUIClient) FetchTransactions(address string) ([]firefly.CustomTransaction, error) {
	result, err := c.makeRPCCall("suix_queryTransactionBlocks",
		[]interface{}{
			map[string]interface{}{
				"filter": map[string]interface{}{
					"ToAddress": address,
				},
			},
			nil,
			50,
			true,
		})
	if err != nil {
		return nil, err
	}

	var txData struct {
		Data []struct {
			Digest    string
			Timestamp string
			Effects   struct {
				Status string
			}
			Transaction struct {
				Data struct {
					Amount float64
				}
			}
		}
	}

	if err := json.Unmarshal(result, &txData); err != nil {
		return nil, fmt.Errorf("failed to parse transactions: %w", err)
	}

	var transactions []firefly.CustomTransaction
	for _, tx := range txData.Data {
		// Skip failed transactions
		if tx.Effects.Status != "success" {
			continue
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, tx.Timestamp)
		if err != nil {
			continue
		}

		transactions = append(transactions, firefly.CustomTransaction{
			ID:          tx.Digest,
			Currency:    "SUI",
			Amount:      tx.Transaction.Data.Amount,
			TransType:   "deposit", // Since we filtered for ToAddress
			Description: fmt.Sprintf("SUI transaction %s", tx.Digest),
			Date:        timestamp,
		})
	}

	return transactions, nil
}

// GetBalance retrieves the current balance for a SUI address
func (c *SUIClient) GetBalance(address string) (float64, error) {
	result, err := c.makeRPCCall("suix_getBalance",
		[]interface{}{
			address,
			"0x2::sui::SUI", // SUI coin type
		})
	if err != nil {
		return 0, err
	}

	var balance struct {
		TotalBalance float64
	}

	if err := json.Unmarshal(result, &balance); err != nil {
		return 0, fmt.Errorf("failed to parse balance: %w", err)
	}

	return balance.TotalBalance, nil
}

// GetName returns the name of the blockchain
func (c *SUIClient) GetName() string {
	return "sui"
}

package blockchain

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/domain/models"
	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	// We might need config later if API keys or specific endpoints are needed
	// "github.com/ZanzyTHEbar/firedragon-go/internal/config"
)

const (
	solanaScanAPIBaseURL = "https://api.solscan.io"
	solNativeMint        = "So11111111111111111111111111111111111111112" // Address for native SOL
)

// SolanaClient implements the BlockchainClient interface for Solana
type SolanaClient struct {
	endpoint   string
	httpClient *http.Client
	// config *config.BlockchainConfig // Add if config needed
}

// NewSolanaClient creates a new Solana client
// func NewSolanaClient(cfg *config.BlockchainConfig) (interfaces.BlockchainClient, error) { // Adjusted signature if config is needed
func NewSolanaClient() (interfaces.BlockchainClient, error) {
	return &SolanaClient{
		endpoint: solanaScanAPIBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		// config: cfg, // Add if config needed
	}, nil
}

// FetchTransactions retrieves transactions for a Solana address using the Solscan API
func (c *SolanaClient) FetchTransactions(address string) ([]models.Transaction, error) {
	// Note: Solscan API might require pagination for full history. This fetches recent ones.
	url := fmt.Sprintf("%s/account/transactions?account=%s&limit=50", c.endpoint, address) // Limit might need adjustment

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, interfaces.NewClientError(interfaces.ErrorTypeNetwork, "failed to create solana request", err)
	}
	// TODO: Add API Key if required by Solscan

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, interfaces.NewClientError(interfaces.ErrorTypeNetwork, "failed to fetch solana transactions", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, interfaces.NewClientError(interfaces.ErrorTypeNetwork, fmt.Sprintf("solana API returned non-200 status: %d", resp.StatusCode), nil)
	}

	// Define struct matching Solscan's transaction response structure
	var result []struct {
		BlockTime int64  `json:"blockTime"`
		Slot      uint64 `json:"slot"`
		TxHash    string `json:"txHash"`
		Fee       uint64 `json:"fee"`
		Status    string `json:"status"`
		Lamport   int64  `json:"lamport"` // Amount in lamports
		Signer    []string `json:"signer"`
		ParsedInstruction []struct {
			ProgramId string `json:"programId"`
			Parsed    struct {
				Info struct {
					Source      string `json:"source"`
					Destination string `json:"destination"`
					Lamports    uint64 `json:"lamports"`
					Amount      string `json:"amount"` // Can be string for SPL tokens
				} `json:"info"`
				Type string `json:"type"`
			} `json:"parsed"`
		} `json:"parsedInstruction"`
		TokenBalanceChange []struct {
			Mint        string  `json:"mint"`
			Amount      float64 `json:"amount"` // Using float for simplicity, might need decimal type
			Decimals    int     `json:"decimals"`
			TokenSymbol string  `json:"tokenSymbol"`
		} `json:"tokenBalanceChange"`
		// Add other fields if needed
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, interfaces.NewClientError(interfaces.ErrorTypeInvalid, "failed to decode solana response", err)
	}

	var transactions []models.Transaction
	for _, tx := range result {
		// Skip failed transactions
		if tx.Status != "Success" {
			continue
		}

		timestamp := time.Unix(tx.BlockTime, 0)
		amount := 0.0
		txType := models.TransactionTypeTransfer // Default, adjust based on context
		description := fmt.Sprintf("Solana Transaction %s", tx.TxHash)
		currency := "SOL" // Default, adjust for SPL tokens

		// Basic logic to determine type and amount (needs refinement for complex txs)
		isSender := false
		isReceiver := false
		for _, signer := range tx.Signer {
			if signer == address {
				isSender = true
				break
			}
		}

		// Check token balance changes first for SPL transfers
		splTransferProcessed := false
		for _, change := range tx.TokenBalanceChange {
			if change.Mint != solNativeMint { // Process SPL tokens
				// This logic is simplified. Real logic needs to check source/dest based on instructions
				if isSender && change.Amount < 0 { // Sent SPL token
					amount = -change.Amount // Make positive for expense/transfer
					currency = change.TokenSymbol
					txType = models.TransactionTypeTransfer // Or Expense if context known
					description = fmt.Sprintf("Sent %f %s", amount, currency)
					splTransferProcessed = true
					break
				} else if !isSender && change.Amount > 0 { // Received SPL token (approximation)
					// Need better logic to confirm receiver based on instructions
					amount = change.Amount
					currency = change.TokenSymbol
					txType = models.TransactionTypeTransfer // Or Income if context known
					description = fmt.Sprintf("Received %f %s", amount, currency)
					splTransferProcessed = true
					break
				}
			}
		}

		// If not an SPL transfer, check native SOL transfer via instructions
		if !splTransferProcessed {
			for _, instruction := range tx.ParsedInstruction {
				// Look for system program transfers
				if instruction.ProgramId == "11111111111111111111111111111111" && instruction.Parsed.Type == "transfer" {
					lamports := instruction.Parsed.Info.Lamports
					solAmount := float64(lamports) / 1e9 // Convert lamports to SOL

					if instruction.Parsed.Info.Source == address {
						isReceiver = false // Confirmed sender
						amount = solAmount
						txType = models.TransactionTypeTransfer // Or Expense
						description = fmt.Sprintf("Sent %f SOL", amount)
						break
					} else if instruction.Parsed.Info.Destination == address {
						isReceiver = true // Confirmed receiver
						amount = solAmount
						txType = models.TransactionTypeTransfer // Or Income
						description = fmt.Sprintf("Received %f SOL", amount)
						break
					}
				}
			}
		}
		
		// If still no amount/type determined, it might be a contract interaction, skip for now
		if amount == 0 {
			continue
		}

		// Determine final type based on sender/receiver status
		if isSender && !isReceiver {
			txType = models.TransactionTypeExpense // Or Transfer if dest known
		} else if !isSender && isReceiver {
			txType = models.TransactionTypeIncome // Or Transfer if source known
		} else {
			// Could be self-transfer or complex interaction, mark as transfer
			txType = models.TransactionTypeTransfer
		}


		transactions = append(transactions, models.Transaction{
			ID:          tx.TxHash, // Use Solscan Tx Hash as unique ID
			Amount:      amount,
			Description: description,
			Date:        timestamp,
			Type:        txType,
			Status:      models.TransactionStatusCompleted, // Assuming success if Status == "Success"
			WalletID:    address, // Associate with the queried wallet
			// CategoryID:  Needs categorization logic
			// DestWalletID: Needs logic to determine for transfers
			CreatedAt:   time.Now(), // Record creation time
			UpdatedAt:   time.Now(),
		})
	}

	return transactions, nil
}

// GetBalance retrieves the current SOL balance for a Solana address using Solscan API
func (c *SolanaClient) GetBalance(address string) (models.BalanceInfo, error) {
	balanceInfo := models.BalanceInfo{Currency: "SOL"} // Default to SOL
	url := fmt.Sprintf("%s/account/%s", c.endpoint, address) // Use account info endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return balanceInfo, interfaces.NewClientError(interfaces.ErrorTypeNetwork, "failed to create solana balance request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return balanceInfo, interfaces.NewClientError(interfaces.ErrorTypeNetwork, "failed to fetch solana balance", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Handle rate limits specifically if possible (e.g., 429 status code)
		return balanceInfo, interfaces.NewClientError(interfaces.ErrorTypeNetwork, fmt.Sprintf("solana balance API returned non-200 status: %d", resp.StatusCode), nil)
	}

	// Define struct matching Solscan's account response structure
	var result struct {
		Data struct {
			Lamports uint64 `json:"lamports"`
			// Other fields like owner, executable, rentEpoch etc.
		} `json:"data"`
		Success bool `json:"success"` // Check if Solscan API provides a success flag
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return balanceInfo, interfaces.NewClientError(interfaces.ErrorTypeInvalid, "failed to decode solana balance response", err)
	}

	// Assuming the endpoint provides success status, check it if available
	// if !result.Success {
	// 	return balanceInfo, interfaces.NewClientError(interfaces.ErrorTypeInvalid, "solana balance API indicated failure", nil)
	// }

	balanceInfo.Amount = float64(result.Data.Lamports) / 1e9 // Convert lamports to SOL

	return balanceInfo, nil
}

// GetChainType returns the name of the blockchain
func (c *SolanaClient) GetChainType() string {
	return "solana"
}

// IsValidAddress validates a Solana wallet address format (basic check)
func (c *SolanaClient) IsValidAddress(address string) bool {
	// Basic validation: Solana addresses are typically base58 encoded strings
	// of a specific length range. This is a very basic check.
	// A proper check would involve base58 decoding and length validation.
	// Example length check (may vary slightly):
	return len(address) >= 32 && len(address) <= 44
}

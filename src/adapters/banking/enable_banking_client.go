package banking

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/firefly"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
)

// EnableBankingClient implements the BankAccountClient interface
type EnableBankingClient struct {
	name         string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	token        string
	tokenExpiry  time.Time
}

// NewEnableBankingClient creates a new Enable Banking client
func NewEnableBankingClient(name, clientID, clientSecret string) *EnableBankingClient {
	return &EnableBankingClient{
		name:         name,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// authenticate gets an access token for the API
func (c *EnableBankingClient) authenticate() error {
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return nil // Token still valid
	}

	data := map[string]string{
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
		"grant_type":    "client_credentials",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %w", err)
	}

	resp, err := c.httpClient.Post(
		"https://auth.enablebanking.com/oauth2/token",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.token = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return nil
}

// FetchBalances retrieves all account balances
func (c *EnableBankingClient) FetchBalances() ([]internal.Balance, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET",
		fmt.Sprintf("https://api.enablebanking.com/v1/accounts/%s/balances", c.name),
		nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch balances: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Balances []struct {
			Amount struct {
				Value    float64
				Currency string
			}
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode balances: %w", err)
	}

	var balances []internal.Balance
	for _, b := range result.Balances {
		balances = append(balances, internal.Balance{
			Currency: b.Amount.Currency,
			Amount:   b.Amount.Value,
		})
	}

	return balances, nil
}

// FetchTransactions retrieves transactions with customization options
func (c *EnableBankingClient) FetchTransactions(limit int, fromDate, toDate string) ([]firefly.CustomTransaction, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.enablebanking.com/v1/accounts/%s/transactions", c.name)
	if fromDate != "" {
		url += fmt.Sprintf("?fromDate=%s", fromDate)
		if toDate != "" {
			url += fmt.Sprintf("&toDate=%s", toDate)
		}
	}
	if limit > 0 {
		url += fmt.Sprintf("&limit=%d", limit)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Transactions []struct {
			ID          string
			BookingDate string
			Amount      struct {
				Value    float64
				Currency string
			}
			CreditDebitIndicator string
			TransactionDetails   struct {
				Description string
			}
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode transactions: %w", err)
	}

	var transactions []firefly.CustomTransaction
	for _, tx := range result.Transactions {
		date, err := time.Parse("2006-01-02", tx.BookingDate)
		if err != nil {
			continue
		}

		transType := "withdrawal"
		if tx.CreditDebitIndicator == "CRDT" {
			transType = "deposit"
		}

		transactions = append(transactions, firefly.CustomTransaction{
			ID:          tx.ID,
			Currency:    tx.Amount.Currency,
			Amount:      tx.Amount.Value,
			TransType:   transType,
			Description: tx.TransactionDetails.Description,
			Date:        date,
		})
	}

	return transactions, nil
}

// GetName returns the name of the bank provider
func (c *EnableBankingClient) GetName() string {
	return "enable_banking"
}

// GetAccountName returns the account identifier
func (c *EnableBankingClient) GetAccountName() string {
	return c.name
}

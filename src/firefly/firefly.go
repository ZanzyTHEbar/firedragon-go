package firefly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FireflyClient represents a client for the Firefly III API
type FireflyClient struct {
	baseURL   string
	token     string
	client    *http.Client
	clientAPI *ClientWithResponses
}

// CustomTransaction represents a financial transaction in our domain model
// Different from the generated Transaction type
type CustomTransaction struct {
	ID          string
	Currency    string
	Amount      float64
	TransType   string // "deposit" or "withdrawal"
	Description string
	Date        time.Time
}

// Balance represents an account balance
type Balance struct {
	Currency string
	Amount   float64
}

// New creates a new FireflyClient
func New(baseURL, token string) (*FireflyClient, error) {
	// Create HTTP client with reasonable timeouts
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create generated API client
	clientAPI, err := NewClientWithResponses(baseURL, WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create Firefly III API client: %w", err)
	}

	return &FireflyClient{
		baseURL:   baseURL,
		token:     token,
		client:    httpClient,
		clientAPI: clientAPI,
	}, nil
}

// CreateTransaction creates a transaction in Firefly III
func (c *FireflyClient) CreateTransaction(accountID, currencyID string, t CustomTransaction) error {
	ctx := context.Background()

	// Determine the transaction type
	var transactionType TransactionTypeProperty
	if t.TransType == "deposit" {
		transactionType = Deposit
	} else {
		transactionType = Withdrawal
	}

	// Create the transaction request
	request := StoreTransactionJSONRequestBody{
		ErrorIfDuplicateHash: new(bool), // Allow duplicates if needed
		ApplyRules:           new(bool), // Apply rules to the transaction
		Transactions: []TransactionSplitStore{
			{
				Type:        transactionType,
				Date:        t.Date,
				Amount:      fmt.Sprintf("%.2f", t.Amount),
				Description: t.Description,
				CurrencyId:  &currencyID,
			},
		},
	}
	*request.ErrorIfDuplicateHash = false
	*request.ApplyRules = true

	// Set source or destination account based on transaction type
	if t.TransType == "deposit" {
		// For deposits, the destination is the account we're importing into
		request.Transactions[0].DestinationId = &accountID
	} else {
		// For withdrawals, the source is the account we're importing from
		request.Transactions[0].SourceId = &accountID
	}

	// Make API request to create transaction with empty params
	params := StoreTransactionParams{}
	resp, err := c.clientAPI.StoreTransactionWithResponse(ctx, &params, request)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode() >= 400 {
		return fmt.Errorf("failed to create transaction (HTTP %d)", resp.StatusCode())
	}

	return nil
}

// GetCurrencyID retrieves the currency ID for an account
func (c *FireflyClient) GetCurrencyID(accountID string) (string, error) {
	ctx := context.Background()

	// Get account details
	params := GetAccountParams{}
	resp, err := c.clientAPI.GetAccountWithResponse(ctx, accountID, &params)
	if err != nil {
		return "", fmt.Errorf("failed to get account details: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode() >= 400 {
		return "", fmt.Errorf("failed to get account (HTTP %d)", resp.StatusCode())
	}

	// Extract currency ID from response
	if resp.HTTPResponse == nil || resp.Body == nil {
		return "", fmt.Errorf("account data not found or malformed")
	}

	// Parse the account data
	var accountSingle AccountSingle
	if err := json.Unmarshal(resp.Body, &accountSingle); err != nil {
		return "", fmt.Errorf("failed to parse account data: %w", err)
	}

	// Check if data exists before accessing fields
	if accountSingle.Data.Attributes.CurrencyId == nil {
		return "", fmt.Errorf("currency ID not found for account")
	}

	return *accountSingle.Data.Attributes.CurrencyId, nil
}

// GetAccountsWithType returns all accounts of a specific type
func (c *FireflyClient) GetAccountsWithType(accountType string) ([]AccountRead, error) {
	ctx := context.Background()

	// Get accounts with optional type filter
	typeFilter := AccountTypeFilter(accountType)
	params := ListAccountParams{
		Type: &typeFilter,
	}

	resp, err := c.clientAPI.ListAccountWithResponse(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("failed to list accounts (HTTP %d)", resp.StatusCode())
	}

	// Parse the account data
	if resp.HTTPResponse == nil || resp.Body == nil {
		return nil, fmt.Errorf("account data not found or malformed")
	}

	var accountArray AccountArray
	if err := json.Unmarshal(resp.Body, &accountArray); err != nil {
		return nil, fmt.Errorf("failed to parse account data: %w", err)
	}

	return accountArray.Data, nil
}

// FetchBalances retrieves account balances
func (c *FireflyClient) FetchBalances() ([]Balance, error) {
	// Get all asset accounts
	accounts, err := c.GetAccountsWithType(string(AccountTypeFilterAsset))
	if err != nil {
		return nil, err
	}

	balances := make([]Balance, 0, len(accounts))
	for _, account := range accounts {
		// Check for required fields using safer approach without nil comparison
		if account.Attributes.CurrentBalance == nil || account.Attributes.CurrencyCode == nil {
			continue
		}

		// Extract balance and currency info, checking for empty values
		if *account.Attributes.CurrentBalance != "" && *account.Attributes.CurrencyCode != "" {
			var amount float64
			if _, err := fmt.Sscanf(*account.Attributes.CurrentBalance, "%f", &amount); err == nil {
				balance := Balance{
					Currency: *account.Attributes.CurrencyCode,
					Amount:   amount,
				}
				balances = append(balances, balance)
			}
		}
	}

	return balances, nil
}

// GetAccounts retrieves all accounts from Firefly III
func (c *FireflyClient) GetAccounts() (map[string]string, error) {
	ctx := context.Background()
	resp, err := c.clientAPI.ListAccountWithResponse(ctx, &ListAccountParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to list accounts (HTTP %d)", resp.StatusCode())
	}

	accounts := make(map[string]string)
	// Parse the response body
	var accountArray AccountArray
	if err := json.NewDecoder(resp.HTTPResponse.Body).Decode(&accountArray); err != nil {
		return nil, fmt.Errorf("failed to parse account data: %w", err)
	}

	for _, acc := range accountArray.Data {
		accounts[acc.Attributes.Name] = acc.Id
	}

	return accounts, nil
}

// Ptr is a helper function to get a pointer to a value
func Ptr[T any](v T) *T {
	return &v
}

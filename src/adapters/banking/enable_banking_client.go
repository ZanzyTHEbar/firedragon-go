package banking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/firefly"
	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
)

const (
	// EnableBankingAuthURL is the OAuth token endpoint
	EnableBankingAuthURL = "https://auth.enablebanking.com/oauth2/token"
	// EnableBankingAPIURL is the base API URL
	EnableBankingAPIURL = "https://api.enablebanking.com/v1"
	// DefaultTimeout for HTTP requests
	DefaultTimeout = 30 * time.Second
	// TokenBufferTime ensures we refresh before expiration
	TokenBufferTime = 5 * time.Minute
	// MaxRetries for transient failures
	MaxRetries = 3
	// ComponentName for logging
	ComponentName internal.Component = "EnableBanking"
)

// RetryBackoff time between retries
var RetryBackoff = 1 * time.Second

// HTTPClient interface for dependency injection and testing
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// LoggerAdapter adapts the internal.Logger to provide simpler logging methods
type LoggerAdapter struct {
	logger *internal.Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger *internal.Logger) *LoggerAdapter {
	if logger == nil {
		logger = internal.GetLogger()
	}
	return &LoggerAdapter{
		logger: logger,
	}
}

// Debug logs a debug message
func (l *LoggerAdapter) Debug(msg string) {
	l.logger.Debug(ComponentName, "%s", msg)
}

// Debugf logs a formatted debug message
func (l *LoggerAdapter) Debugf(format string, args ...interface{}) {
	l.logger.Debug(ComponentName, format, args...)
}

// Info logs an info message
func (l *LoggerAdapter) Info(msg string) {
	l.logger.Info(ComponentName, "%s", msg)
}

// Infof logs a formatted info message
func (l *LoggerAdapter) Infof(format string, args ...interface{}) {
	l.logger.Info(ComponentName, format, args...)
}

// Warn logs a warning message
func (l *LoggerAdapter) Warn(msg string) {
	l.logger.Warn(ComponentName, "%s", msg)
}

// Warnf logs a formatted warning message
func (l *LoggerAdapter) Warnf(format string, args ...interface{}) {
	l.logger.Warn(ComponentName, format, args...)
}

// Error logs an error message
func (l *LoggerAdapter) Error(msg string) {
	l.logger.Error(ComponentName, "%s", msg)
}

// Errorf logs a formatted error message
func (l *LoggerAdapter) Errorf(format string, args ...interface{}) {
	l.logger.Error(ComponentName, format, args...)
}

// EnableBankingClient implements the BankAccountClient interface
type EnableBankingClient struct {
	name         string
	clientID     string
	clientSecret string
	redirectURI  string
	httpClient   HTTPClient
	token        string
	tokenExpiry  time.Time
	mu           sync.RWMutex // Protects token data from concurrent access
	logger       *LoggerAdapter
	authURL      string
	apiURL       string
	timeout      time.Duration
}

// EnableBankingConfig holds the configuration for the Enable Banking client
type EnableBankingConfig struct {
	Name         string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Timeout      time.Duration
	Logger       *internal.Logger
	HTTPClient   HTTPClient
	AuthURL      string
	APIURL       string
}

// NewEnableBankingClient creates a new Enable Banking client
func NewEnableBankingClient(config EnableBankingConfig) interfaces.BankAccountClient {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	loggerAdapter := NewLoggerAdapter(config.Logger)

	// Use provided HTTP client or create a default one
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	// Use provided URLs or use defaults
	authURL := config.AuthURL
	if authURL == "" {
		authURL = EnableBankingAuthURL
	}

	apiURL := config.APIURL
	if apiURL == "" {
		apiURL = EnableBankingAPIURL
	}
	return &EnableBankingClient{
		name:         config.Name,
		clientID:     config.ClientID,
		clientSecret: config.ClientSecret,
		redirectURI:  config.RedirectURI,
		httpClient:   httpClient,
		logger:       loggerAdapter,
		authURL:      authURL,
		apiURL:       apiURL,
		timeout:      timeout,
	}
}

// authenticate gets an access token for the API with retry logic
func (c *EnableBankingClient) authenticate() error {
	c.mu.RLock()
	if c.token != "" && time.Now().Add(TokenBufferTime).Before(c.tokenExpiry) {
		c.mu.RUnlock()
		return nil // Token still valid with buffer time
	}
	c.mu.RUnlock()

	// Need a new token
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.token != "" && time.Now().Add(TokenBufferTime).Before(c.tokenExpiry) {
		return nil // Another goroutine already refreshed the token
	}

	data := map[string]string{
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
		"grant_type":    "client_credentials",
	}

	if c.redirectURI != "" {
		data["redirect_uri"] = c.redirectURI
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %w", err)
	}

	var tokenResponse TokenResponse
	var lastErr error

	// Implement retry logic for authentication
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Infof("Retrying authentication (attempt %d of %d)", attempt, MaxRetries)
			time.Sleep(RetryBackoff * time.Duration(attempt))
		}

		req, err := http.NewRequest(
			http.MethodPost,
			c.authURL,
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			lastErr = fmt.Errorf("failed to create auth request: %w", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("auth request failed: %w", err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errResp struct {
				Error            string `json:"error"`
				ErrorDescription string `json:"error_description"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
				lastErr = fmt.Errorf("auth failed: %s - %s (HTTP %d)",
					errResp.Error, errResp.ErrorDescription, resp.StatusCode)
			} else {
				lastErr = fmt.Errorf("auth failed with status: %s (HTTP %d)",
					resp.Status, resp.StatusCode)
			}
			continue
		}

		if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
			lastErr = fmt.Errorf("failed to decode auth response: %w", err)
			continue
		}

		// Success! Update token information
		c.token = tokenResponse.AccessToken
		c.tokenExpiry = time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
		c.logger.Debugf("Token refreshed, expires in %d seconds", tokenResponse.ExpiresIn)
		return nil
	}

	return fmt.Errorf("authentication failed after %d attempts: %w", MaxRetries, lastErr)
}

// makeRequest is a helper function to make authenticated API requests with retries
func (c *EnableBankingClient) makeRequest(ctx context.Context, method, path string, query map[string]string) (*http.Response, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s", c.apiURL, path)

	// Add query parameters if any
	if len(query) > 0 {
		url += "?"
		first := true
		for k, v := range query {
			if !first {
				url += "&"
			}
			url += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
	}

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Infof("Retrying request to %s (attempt %d of %d)", path, attempt, MaxRetries)
			time.Sleep(RetryBackoff * time.Duration(attempt))
		}

		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		c.mu.RLock()
		token := c.token
		c.mu.RUnlock()

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err = c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		// Handle token expiration by refreshing and retrying
		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			c.mu.Lock()
			c.token = "" // Force token refresh
			c.mu.Unlock()
			if err := c.authenticate(); err != nil {
				lastErr = fmt.Errorf("failed to refresh token: %w", err)
				continue
			}
			continue // Retry with new token
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("API request failed: status=%d body=%s", resp.StatusCode, body.String())
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", MaxRetries, lastErr)
}

// FetchBalances retrieves all account balances
func (c *EnableBankingClient) FetchBalances() ([]interfaces.Balance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	path := fmt.Sprintf("/accounts/%s/balances", c.name)
	resp, err := c.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch balances: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Balances []struct {
			BalanceType string `json:"balanceType"`
			Amount      struct {
				Value    float64 `json:"value"`
				Currency string  `json:"currency"`
			} `json:"amount"`
			CreditDebitIndicator string    `json:"creditDebitIndicator"`
			LastChangeDateTime   time.Time `json:"lastChangeDateTime"`
		} `json:"balances"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode balances response: %w", err)
	}

	var balances []interfaces.Balance
	for _, b := range result.Balances {
		balances = append(balances, interfaces.Balance{
			Currency: b.Amount.Currency,
			Amount:   b.Amount.Value,
		})
	}

	c.logger.Infof("Retrieved %d balance(s) for account %s", len(balances), c.name)
	return balances, nil
}

// FetchTransactions retrieves transactions with customization options
func (c *EnableBankingClient) FetchTransactions(limit int, fromDate, toDate string) ([]firefly.CustomTransaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	path := fmt.Sprintf("/accounts/%s/transactions", c.name)
	query := make(map[string]string)

	if fromDate != "" {
		query["fromBookingDate"] = fromDate
	}

	if toDate != "" {
		query["toBookingDate"] = toDate
	}

	if limit > 0 {
		query["limit"] = fmt.Sprintf("%d", limit)
	}

	resp, err := c.makeRequest(ctx, http.MethodGet, path, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Transactions []struct {
			ID          string `json:"transactionId"`
			BookingDate string `json:"bookingDate"`
			ValueDate   string `json:"valueDate"`
			Amount      struct {
				Value    float64 `json:"value"`
				Currency string  `json:"currency"`
			} `json:"transactionAmount"`
			CreditDebitIndicator string `json:"creditDebitIndicator"`
			TransactionDetails   struct {
				Description           string `json:"description"`
				MerchantName          string `json:"merchantName,omitempty"`
				RemittanceInformation string `json:"remittanceInformation,omitempty"`
				Category              string `json:"category,omitempty"`
			} `json:"transactionDetails"`
			Status string `json:"status"`
		} `json:"transactions"`
		PaginationInfo struct {
			TotalItems  int `json:"totalItems"`
			TotalPages  int `json:"totalPages"`
			CurrentPage int `json:"currentPage"`
			PageSize    int `json:"pageSize"`
		} `json:"paginationInfo"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode transactions response: %w", err)
	}

	var transactions []firefly.CustomTransaction
	for _, tx := range result.Transactions {
		// Skip transactions that are not booked
		if tx.Status != "BOOK" {
			continue
		}

		date, err := time.Parse("2006-01-02", tx.BookingDate)
		if err != nil {
			c.logger.Warnf("Failed to parse booking date: %s for transaction %s", tx.BookingDate, tx.ID)
			continue
		}

		transType := "withdrawal"
		if tx.CreditDebitIndicator == "CRDT" {
			transType = "deposit"
		}

		// Build a more complete description
		description := tx.TransactionDetails.Description
		if description == "" {
			if tx.TransactionDetails.MerchantName != "" {
				description = tx.TransactionDetails.MerchantName
			} else if tx.TransactionDetails.RemittanceInformation != "" {
				description = tx.TransactionDetails.RemittanceInformation
			}
		}

		// Ensure we have at least some description
		if description == "" {
			description = fmt.Sprintf("Transaction %s", tx.ID)
		}

		transactions = append(transactions, firefly.CustomTransaction{
			ID:          tx.ID,
			Currency:    tx.Amount.Currency,
			Amount:      tx.Amount.Value,
			TransType:   transType,
			Description: description,
			Date:        date,
		})
	}

	c.logger.Infof("Retrieved %d transaction(s) for account %s", len(transactions), c.name)
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

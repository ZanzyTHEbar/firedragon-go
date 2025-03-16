// filepath: /mnt/dragonnet/common/Projects/Personal/OpenSource/firedragon-go/src/adapters/banking/enable_banking_client_test.go
package banking

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/internal"
)

// MockHTTPClient implements HTTPClient for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do implements HTTPClient interface
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// TestEnableBankingClient_Authentication tests the authentication flow
func TestEnableBankingClient_Authentication(t *testing.T) {
	testCases := []struct {
		name         string
		clientID     string
		clientSecret string
		authResponse *http.Response
		authError    error
		expectError  bool
	}{
		{
			name:         "successful authentication",
			clientID:     "valid_client_id",
			clientSecret: "valid_client_secret",
			authResponse: &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(`{
					"access_token": "test_access_token",
					"expires_in": 3600,
					"token_type": "Bearer"
				}`)),
			},
			authError:   nil,
			expectError: false,
		},
		{
			name:         "authentication fails with invalid credentials",
			clientID:     "invalid_client_id",
			clientSecret: "invalid_client_secret",
			authResponse: &http.Response{
				StatusCode: 401,
				Body: io.NopCloser(strings.NewReader(`{
					"error": "invalid_client",
					"error_description": "Invalid client credentials"
				}`)),
			},
			authError:   nil,
			expectError: true,
		},
		{
			name:         "authentication fails with network error",
			clientID:     "valid_client_id",
			clientSecret: "valid_client_secret",
			authResponse: nil,
			authError:    io.EOF,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					// Validate request to auth endpoint
					if !strings.Contains(req.URL.String(), "oauth2/token") {
						t.Errorf("Expected request to auth endpoint, got %s", req.URL.String())
					}

					// Verify content type
					contentType := req.Header.Get("Content-Type")
					if contentType != "application/json" {
						t.Errorf("Expected Content-Type application/json, got %s", contentType)
					}

					// Check request body
					bodyBytes, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("Failed to read request body: %v", err)
					}

					// Reset body for further reads
					req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

					var requestData map[string]string
					if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
						t.Fatalf("Failed to parse request body: %v", err)
					}

					// Validate request data
					if requestData["client_id"] != tc.clientID {
						t.Errorf("Expected client_id %s, got %s", tc.clientID, requestData["client_id"])
					}
					if requestData["client_secret"] != tc.clientSecret {
						t.Errorf("Expected client_secret %s, got %s", tc.clientSecret, requestData["client_secret"])
					}
					if requestData["grant_type"] != "client_credentials" {
						t.Errorf("Expected grant_type client_credentials, got %s", requestData["grant_type"])
					}

					return tc.authResponse, tc.authError
				},
			}

			// Create client
			client := NewEnableBankingClient(EnableBankingConfig{
				Name:         "test_account",
				ClientID:     tc.clientID,
				ClientSecret: tc.clientSecret,
				Logger:       internal.GetLogger(),
				HTTPClient:   mockClient,
			}).(*EnableBankingClient)

			// Test authentication
			err := client.authenticate()
			if tc.expectError && err == nil {
				t.Errorf("Expected authentication error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no authentication error but got: %v", err)
			}

			// Check if token was set correctly for successful auth
			if !tc.expectError {
				if client.token != "test_access_token" {
					t.Errorf("Expected token 'test_access_token' but got '%s'", client.token)
				}

				// Check token expiry
				expectedExpiry := time.Now().Add(time.Hour)
				if client.tokenExpiry.Sub(expectedExpiry) > time.Minute {
					t.Errorf("Token expiry time is not as expected")
				}
			}
		})
	}
}

// TestEnableBankingClient_FetchBalances tests balance retrieval
func TestEnableBankingClient_FetchBalances(t *testing.T) {
	// Mock HTTP responses
	mockHTTP := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Authentication request
			if strings.Contains(req.URL.String(), "oauth2/token") {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "test_access_token",
						"expires_in": 3600,
						"token_type": "Bearer"
					}`)),
				}, nil
			}

			// Balances request
			if strings.Contains(req.URL.String(), "/accounts/test_account/balances") {
				// Check auth header
				authHeader := req.Header.Get("Authorization")
				if authHeader != "Bearer test_access_token" {
					t.Errorf("Expected Authorization header 'Bearer test_access_token', got '%s'", authHeader)
				}

				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"balances": [
							{
								"balanceType": "interimAvailable",
								"amount": {
									"value": 1000.50,
									"currency": "EUR"
								},
								"creditDebitIndicator": "CRDT",
								"lastChangeDateTime": "2023-01-01T00:00:00Z"
							},
							{
								"balanceType": "interimAvailable",
								"amount": {
									"value": 200.25,
									"currency": "USD"
								},
								"creditDebitIndicator": "CRDT",
								"lastChangeDateTime": "2023-01-01T00:00:00Z"
							}
						]
					}`)),
				}, nil
			}

			t.Errorf("Unexpected request to URL: %s", req.URL.String())
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader(`{"error": "Not found"}`)),
			}, nil
		},
	}

	// Create client
	client := NewEnableBankingClient(EnableBankingConfig{
		Name:         "test_account",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		Logger:       internal.GetLogger(),
		HTTPClient:   mockHTTP,
	})

	// Test fetch balances
	balances, err := client.FetchBalances()
	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	// Verify balances
	if len(balances) != 2 {
		t.Errorf("Expected 2 balances but got %d", len(balances))
	}

	expectedBalances := map[string]float64{
		"EUR": 1000.50,
		"USD": 200.25,
	}

	for _, balance := range balances {
		expectedAmount, exists := expectedBalances[balance.Currency]
		if !exists {
			t.Errorf("Unexpected currency: %s", balance.Currency)
			continue
		}

		if balance.Amount != expectedAmount {
			t.Errorf("Expected amount %f for %s but got %f",
				expectedAmount, balance.Currency, balance.Amount)
		}
	}
}

// TestEnableBankingClient_FetchTransactions tests transaction retrieval
func TestEnableBankingClient_FetchTransactions(t *testing.T) {
	// Mock HTTP responses
	mockHTTP := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Authentication request
			if strings.Contains(req.URL.String(), "oauth2/token") {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "test_access_token",
						"expires_in": 3600,
						"token_type": "Bearer"
					}`)),
				}, nil
			}

			// Transactions request
			if strings.Contains(req.URL.String(), "/accounts/test_account/transactions") {
				// Check auth header
				authHeader := req.Header.Get("Authorization")
				if authHeader != "Bearer test_access_token" {
					t.Errorf("Expected Authorization header 'Bearer test_access_token', got '%s'", authHeader)
				}

				// Check query parameters
				if req.URL.Query().Get("limit") != "10" {
					t.Errorf("Expected limit parameter '10', got '%s'", req.URL.Query().Get("limit"))
				}
				if req.URL.Query().Get("fromBookingDate") != "2023-01-01" {
					t.Errorf("Expected fromBookingDate parameter '2023-01-01', got '%s'", req.URL.Query().Get("fromBookingDate"))
				}
				if req.URL.Query().Get("toBookingDate") != "2023-01-31" {
					t.Errorf("Expected toBookingDate parameter '2023-01-31', got '%s'", req.URL.Query().Get("toBookingDate"))
				}

				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"transactions": [
							{
								"transactionId": "tx-001",
								"bookingDate": "2023-01-05",
								"valueDate": "2023-01-06",
								"transactionAmount": {
									"value": -50.25,
									"currency": "EUR"
								},
								"creditDebitIndicator": "DBIT",
								"transactionDetails": {
									"description": "Grocery Store",
									"merchantName": "SuperMarket",
									"category": "Food"
								},
								"status": "BOOK"
							},
							{
								"transactionId": "tx-002",
								"bookingDate": "2023-01-10",
								"valueDate": "2023-01-11",
								"transactionAmount": {
									"value": 1200.00,
									"currency": "EUR"
								},
								"creditDebitIndicator": "CRDT",
								"transactionDetails": {
									"description": "",
									"merchantName": "",
									"remittanceInformation": "Monthly salary payment",
									"category": "Income"
								},
								"status": "BOOK"
							},
							{
								"transactionId": "tx-003",
								"bookingDate": "invalid-date",
								"valueDate": "2023-01-15",
								"transactionAmount": {
									"value": 300.00,
									"currency": "EUR"
								},
								"creditDebitIndicator": "CRDT",
								"transactionDetails": {},
								"status": "BOOK"
							},
							{
								"transactionId": "tx-004",
								"bookingDate": "2023-01-20",
								"valueDate": "2023-01-21",
								"transactionAmount": {
									"value": -75.50,
									"currency": "EUR"
								},
								"creditDebitIndicator": "DBIT",
								"transactionDetails": {},
								"status": "PDNG"
							}
						],
						"paginationInfo": {
							"totalItems": 4,
							"totalPages": 1,
							"currentPage": 1,
							"pageSize": 10
						}
					}`)),
				}, nil
			}

			t.Errorf("Unexpected request to URL: %s", req.URL.String())
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader(`{"error": "Not found"}`)),
			}, nil
		},
	}

	// Create client
	client := NewEnableBankingClient(EnableBankingConfig{
		Name:         "test_account",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		Logger:       internal.GetLogger(),
		HTTPClient:   mockHTTP,
	})

	// Test fetch transactions
	transactions, err := client.FetchTransactions(10, "2023-01-01", "2023-01-31")
	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	// We should have 2 valid transactions
	if len(transactions) != 2 {
		t.Errorf("Expected 2 transactions but got %d", len(transactions))
	}

	// Check first transaction
	if len(transactions) > 0 {
		tx := transactions[0]
		if tx.ID != "tx-001" {
			t.Errorf("Expected transaction ID 'tx-001' but got '%s'", tx.ID)
		}
		if tx.Amount != -50.25 {
			t.Errorf("Expected amount -50.25 but got %f", tx.Amount)
		}
		if tx.Currency != "EUR" {
			t.Errorf("Expected currency EUR but got %s", tx.Currency)
		}
		if tx.Description != "Grocery Store" {
			t.Errorf("Expected description 'Grocery Store' but got '%s'", tx.Description)
		}
		if tx.TransType != "withdrawal" {
			t.Errorf("Expected transaction type 'withdrawal' but got '%s'", tx.TransType)
		}

		expectedDate := time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)
		if !tx.Date.Equal(expectedDate) {
			t.Errorf("Expected date %v but got %v", expectedDate, tx.Date)
		}
	}

	// Check second transaction (with no description but remittance info)
	if len(transactions) > 1 {
		tx := transactions[1]
		if tx.ID != "tx-002" {
			t.Errorf("Expected transaction ID 'tx-002' but got '%s'", tx.ID)
		}
		if tx.Description != "Monthly salary payment" {
			t.Errorf("Expected description from remittanceInformation but got '%s'", tx.Description)
		}
		if tx.TransType != "deposit" {
			t.Errorf("Expected transaction type 'deposit' but got '%s'", tx.TransType)
		}
	}
}

// TestEnableBankingClient_RetryLogic tests the retry mechanism
func TestEnableBankingClient_RetryLogic(t *testing.T) {
	attempts := 0
	// Mock HTTP client that fails twice then succeeds
	mockHTTP := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "oauth2/token") {
				attempts++
				if attempts < 3 {
					// Fail first two attempts
					return &http.Response{
						StatusCode: 500,
						Body:       io.NopCloser(strings.NewReader(`{"error": "Internal Server Error"}`)),
					}, nil
				}

				// Succeed on third attempt
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "test_access_token",
						"expires_in": 3600,
						"token_type": "Bearer"
					}`)),
				}, nil
			}

			t.Errorf("Unexpected request to URL: %s", req.URL.String())
			return nil, io.EOF
		},
	}

	// Override retry backoff for faster tests
	origBackoff := RetryBackoff
	RetryBackoff = 10 * time.Millisecond
	defer func() { RetryBackoff = origBackoff }()

	// Create client
	client := NewEnableBankingClient(EnableBankingConfig{
		Name:         "test_account",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		Logger:       internal.GetLogger(),
		HTTPClient:   mockHTTP,
	}).(*EnableBankingClient)

	// Test authentication with retries
	err := client.authenticate()
	if err != nil {
		t.Errorf("Expected successful authentication after retries but got: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 authentication attempts but got %d", attempts)
	}

	if client.token != "test_access_token" {
		t.Errorf("Expected token 'test_access_token' but got '%s'", client.token)
	}
}

// TestEnableBankingClient_GetName tests the GetName method
func TestEnableBankingClient_GetName(t *testing.T) {
	client := NewEnableBankingClient(EnableBankingConfig{
		Name:   "test_account",
		Logger: internal.GetLogger(),
	})

	name := client.GetName()
	if name != "enable_banking" {
		t.Errorf("Expected provider name 'enable_banking' but got '%s'", name)
	}
}

// TestEnableBankingClient_GetAccountName tests the GetAccountName method
func TestEnableBankingClient_GetAccountName(t *testing.T) {
	client := NewEnableBankingClient(EnableBankingConfig{
		Name:   "test_account",
		Logger: internal.GetLogger(),
	})

	accountName := client.GetAccountName()
	if accountName != "test_account" {
		t.Errorf("Expected account name 'test_account' but got '%s'", accountName)
	}
}

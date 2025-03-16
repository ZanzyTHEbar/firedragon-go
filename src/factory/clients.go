package factory

import (
	"os"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/adapters/banking"
	"github.com/ZanzyTHEbar/firedragon-go/adapters/blockchain"
	"github.com/ZanzyTHEbar/firedragon-go/firefly"
	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
)

func NewDatabaseClient(path string) (interfaces.DatabaseClient, error) {
	return internal.NewSQLiteDatabase(path)
}

// NewBlockchainClient creates a blockchain client based on the chain type
func NewBlockchainClient(chain string) interfaces.BlockchainClient {
	switch chain {
	case "ethereum":
		return blockchain.NewEthereumClient(os.Getenv("ETHERSCAN_API_KEY"))
	case "solana":
		return blockchain.NewSolanaClient()
	case "sui":
		return blockchain.NewSUIClient()
	default:
		return nil
	}
}

// NewBankingClient creates a banking client based on the provider
func NewBankingClient(provider, name string, options map[string]string) interfaces.BankAccountClient {
	logger := internal.GetLogger()
	
	switch provider {
	case "enable_banking":
		clientID := options["client_id"]
		if clientID == "" {
			clientID = os.Getenv("ENABLE_CLIENT_ID")
		}
		
		clientSecret := options["client_secret"]
		if clientSecret == "" {
			clientSecret = os.Getenv("ENABLE_CLIENT_SECRET")
		}
		
		redirectURI := options["redirect_uri"]
		if redirectURI == "" {
			redirectURI = os.Getenv("ENABLE_REDIRECT_URI")
		}
		
		// Parse timeout if provided, otherwise use default
		var timeout time.Duration
		if timeoutStr := options["timeout"]; timeoutStr != "" {
			if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
				timeout = parsedTimeout
			} else {
				logger.Warn(internal.ComponentGeneral, "Invalid timeout value: %s, using default", timeoutStr)
			}
		}
		
		return banking.NewEnableBankingClient(banking.EnableBankingConfig{
			Name:         name,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURI:  redirectURI,
			Timeout:      timeout,
			Logger:       logger,
		})
	default:
		logger.Warn(internal.ComponentGeneral, "Unknown banking provider: %s", provider)
		return nil
	}
}

// NewFireflyClient creates a new Firefly III client
func NewFireflyClient(baseURL, token string) (interfaces.FireflyClient, error) {
	return firefly.New(baseURL, token)
}

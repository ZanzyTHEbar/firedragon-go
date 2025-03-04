package factory

import (
	"os"

	"github.com/ZanzyTHEbar/firedragon-go/adapters/banking"
	"github.com/ZanzyTHEbar/firedragon-go/adapters/blockchain"
	"github.com/ZanzyTHEbar/firedragon-go/firefly"
	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
)

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
func NewBankingClient(provider, name, clientID, clientSecret string) interfaces.BankAccountClient {
	switch provider {
	case "enable_banking":
		return banking.NewEnableBankingClient(name, clientID, clientSecret)
	default:
		return nil
	}
}

// NewFireflyClient creates a new Firefly III client
func NewFireflyClient(baseURL, token string) (interfaces.FireflyClient, error) {
	return firefly.New(baseURL, token)
}

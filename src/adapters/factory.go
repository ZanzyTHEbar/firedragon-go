package adapters

import (
	"os"

	"github.com/ZanzyTHEbar/firedragon-go/adapters/banking"
	"github.com/ZanzyTHEbar/firedragon-go/adapters/blockchain"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
)

// NewBlockchainClient creates a new blockchain client based on chain type
func NewBlockchainClient(chain string) internal.BlockchainClient {
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

// NewBankingClient creates a new banking client based on provider
func NewBankingClient(provider, name, clientID, clientSecret string) internal.BankAccountClient {
	switch provider {
	case "enable_banking":
		return banking.NewEnableBankingClient(name, clientID, clientSecret)
	default:
		return nil
	}
}

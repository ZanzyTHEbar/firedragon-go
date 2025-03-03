# Firedragon

Crypto Wallet and Bank Account Transaction Importer for Firefly III

![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![License](https://img.shields.io/badge/license-MIT-blue)
![Go](https://img.shields.io/badge/Go-1.21-blue)

This project is a Golang-based application designed to integrate cryptocurrency wallets and bank accounts with [Firefly III](https://www.firefly-iii.org/), a self-hosted personal finance management tool. It fetches transaction data from cryptocurrency wallets (e.g., MetaMask on Ethereum, Phantom on Solana, SUIWallet on SUI) and bank accounts (e.g., Revolut via Enable Banking API) and imports it into Firefly III for unified financial tracking.

The aim of this project is to leverage unified API's, such as Enable Banking, to simplify the process of importing transactions from various sources into Firefly III.

I built this as a personal project to make integrating my crypto wallets and bank accounts with Firefly III easier.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

---

## Prerequisites

Before you begin, ensure you have the following installed:

- [Docker](https://www.docker.com/get-started) - For containerized deployment.
- [Docker Compose](https://docs.docker.com/compose/install/) - For managing multi-container setups (optional).
- [Go](https://golang.org/dl/) (version 1.21 or later) - Required if building from source.
- A running instance of [Firefly III](https://docs.firefly-iii.org/installation/docker/) - The target financial management tool.

---

## Installation

Follow these steps to set up the project:

1. **Clone the Repository**:
   ```sh
   git clone https://github.com/ZanzyTHEbar/firedragon-go.git
   cd firedragon-go
   ```

2. **Build the Docker Image**:
   ```sh
   docker build -t firedragon-go .
   ```

3. **(Optional) Use Docker Compose**:
   If using Docker Compose, ensure your `docker-compose.yml` file is properly configured, then proceed to the [Usage](#usage) section.

---

## Configuration

1. **Create a `config.json` File**:
   Place a `config.json` file in the project root with the following structure:
   ```json
   {
       "firefly": {
           "url": "http://localhost:8080",
           "token": "your_firefly_api_token"
       },
       "wallets": [
           {"chain": "ethereum", "address": "0xYourEthAddress"},
           {"chain": "solana", "address": "YourSolanaAddress"},
           {"chain": "sui", "address": "YourSuiAddress"}
       ],
       "banks": [
           {"provider": "revolut", "account_id": "your_revolut_account_id"}
       ],
       "interval": "15m"
   }
   ```
   - `firefly.url`: Your Firefly III instance URL.
   - `firefly.token`: Your Firefly III API token.
   - `wallets`: List of cryptocurrency wallet addresses to track.
   - `banks`: List of bank accounts to import (e.g., Revolut).
   - `interval`: Frequency of transaction imports (e.g., "15m" for 15 minutes).

2. **Set Environment Variables**:
   For sensitive credentials, use environment variables:
   - `ENABLE_CLIENT_ID`: Your Enable Banking OAuth client ID.
   - `ENABLE_CLIENT_SECRET`: Your Enable Banking OAuth client secret.
   - `ETHERSCAN_API_KEY`: Your Etherscan API key (for Ethereum transactions).

   You can define these in a `.env` file or export them in your shell:
   ```sh
   export ENABLE_CLIENT_ID="your_client_id"
   export ENABLE_CLIENT_SECRET="your_client_secret"
   export ETHERSCAN_API_KEY="your_etherscan_key"
   ```

---

## Usage

### Running in Foreground Mode

For testing or interactive use:

```sh
docker run -it --rm -v $(pwd)/data:/app/data firedragon-go --foreground
```

- `-it`: Runs the container interactively.
- `--rm`: Removes the container after it exits.
- `-v`: Mounts a local `data` directory for persistent storage.

### Running in Detached Mode

To run as a background service:

#### With Docker Compose:
```sh
docker-compose up -d
```

#### Manually:
```sh
docker run -d -v $(pwd)/data:/app/data firedragon-go --detach
```

- `-d`: Runs the container in detached mode.

### Stopping the Application

#### With Docker Compose:
```sh
docker-compose down
```

#### Manually:
```sh
docker stop <container_id>
```
Find the `container_id` using `docker ps`.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

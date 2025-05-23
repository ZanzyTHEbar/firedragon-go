# Engineering Design Document: Crypto Wallet and Bank Account Transaction Importer for Firefly III

**Project Name:** Crypto Wallet and Bank Account Transaction Importer  
**Date:** March 3, 2025  
**Version:** 1.2  
**Author:** Zacariah Heim  
**Project Lead:** Zacariah Heim 

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [System Overview](#2-system-overview)
3. [Architecture](#3-architecture)
4. [Components](#4-components)
5. [Interfaces](#5-interfaces)
6. [Data Management](#6-data-management)
7. [Error Handling and Logging](#7-error-handling-and-logging)
8. [Security Considerations](#8-security-considerations)
9. [Testing Strategy](#9-testing-strategy)
10. [Deployment and Maintenance](#10-deployment-and-maintenance)
11. [Conclusion](#11-conclusion)
12. [Appendices](#12-appendices)

---

## 1. Introduction

### 1.1 Purpose

This document outlines the design of a Golang-based application that integrates cryptocurrency wallets and bank accounts with **Firefly III**, a self-hosted personal finance management tool. The application fetches transaction data from cryptocurrency wallets (e.g., MetaMask on Ethereum, Phantom on Solana, SUIWallet on SUI) and bank accounts (e.g., Revolut via Enable Banking API) and imports it into Firefly III for unified financial tracking. It is designed to run as a background service within a Docker container, managed by Docker Compose, ensuring ease of deployment and operation.

### 1.2 Scope

The application will:

- Retrieve transaction data from specified cryptocurrency wallets and bank accounts.
- Support multiple blockchain networks and banks through modular adapters.
- Integrate with Firefly III's REST API to create and manage transactions.
- Use Pocketbase v0.26.1 with custom database driver to track imported transactions and prevent duplicates.
  - Support customizable transaction fetching for bank accounts (e.g., limit, date range).
  - Track imported transactions in SQLite to prevent duplicates.
  - Enable concurrent fetching and processing for multiple wallets and accounts.
  - Use Pocketbase to handle Authentication and Authorization for the API.
  - Use Pocketbase to handle the database and data management.
  - Use Pocketbase to handle the API and data management.
- Use the Enable Banking API to fetch transaction and balance data from bank accounts. 
- Enable concurrent data fetching and processing for efficiency.
- Run as a background service in detached mode, with signal handling for graceful shutdown.
  - We will run on linux only, for now, and will not support Windows or Mac.
  - We will use a client & daemon architecture.
  - We will use use linux kernel APIs to manage the daemon.
  - We will use proper privilage separation to ensure the daemon runs with the least amount of privileges. And the client runs in user-space.
  - We will have systemd services to manage the daemon & the client.
  - We will use linux IPC to communicate between the client & the daemon.
- Provide a command-line interface (CLI) for user interaction, supporting both foreground and detached modes.
- Be configurable via a TOML file, allowing users to specify wallet addresses, bank account details, and Firefly III API token. 
- Be packaged as a custom Docker image with Docker Compose for self-hosting.
- Be designed with a hexagonal architecture, employing Dependency Injection (DI) and Inversion of Control (IoC) principles to ensure modularity and testability.

### 1.3 Definitions and Acronyms

- **API**: Application Programming Interface
- **ASPSP**: Account Servicing Payment Service Provider (e.g., Revolut)
- **CLI**: Command-Line Interface
- **DI**: Dependency Injection
- **IoC**: Inversion of Control
- **OAS**: Open API Specification
- **SOLID**: Software design principles (Single Responsibility, Open/Closed, etc.)
- **SQLite**: Lightweight, embeddable SQL database

---

## 2. System Overview

### 2.1 System Context

The application is a standalone CLI tool designed to operate as a background service within a Docker container. It interacts with external blockchain APIs (e.g., Etherscan, Solscan, SUI JSON-RPC), the Enable Banking API for bank data, and Firefly III's REST API. Configuration is read from a TOML file, and state is managed using Pocketbase. Docker Compose orchestrates the application and its dependencies, enabling self-hosting on any Docker-compatible instance.

### 2.2 Functional Requirements

- Fetch transaction data from cryptocurrency wallets on Ethereum, Solana, and SUI blockchains.
- Fetch transaction and balance data from bank accounts via Enable Banking API, supporting multiple banks (starting with Revolut).
- Integrate fetched data into Firefly III by creating corresponding transactions.
- Support customizable transaction fetching for bank accounts (e.g., limit, date range).
- Track imported transactions in SQLite to prevent duplicates.
- Enable concurrent fetching and processing for multiple wallets and accounts.
- Run as a background service in detached mode, with signal handling for graceful shutdown.
- Provide a CLI interface with foreground and detached modes.

### 2.3 Non-Functional Requirements

- **Performance**: Efficiently process up to 10,000 transactions per wallet or account.
- **Scalability**: Support easy addition of new blockchains and banks.
- **Reliability**: Gracefully handle errors (e.g., network issues, API failures) with user feedback.
- **Security**: Protect credentials and sensitive data from unauthorized access.
- **Portability**: Ensure consistent operation across environments via Docker.

---

## 3. Architecture

### 3.1 Architectural Style

The system employs a **hexagonal architecture** (ports and adapters) to isolate core business logic from external systems, promoting flexibility and maintainability. **Dependency Injection (DI)** and **Inversion of Control (IoC)** decouple components, adhering to **SOLID principles**. The application is containerized with Docker, and Docker Compose manages deployment.

### 3.2 System Components

- **Configuration**: TOML file for settings, wallet addresses, bank account details, and mappings.
- **Database**: SQLite for tracking imported transactions, persisted via Docker volume.
- **Blockchain Clients**: Adapters for fetching cryptocurrency transaction data.
- **Bank Account Clients**: Adapters for fetching bank transaction and balance data via Enable Banking API.
- **Firefly III Client**: Adapter for interacting with Firefly III's API.
- **Services**: Core logic for transaction fetching, parsing, and importing.
- **Service Manager**: Manages goroutine lifecycles for concurrent operations and background tasks.
- **CLI**: Command-line interface supporting foreground and detached modes.

### 3.3 Component Diagram

```plaintext
+--------------------+
|  Configuration     |
+--------------------+
           |
           v
+--------------------+
|  Service Manager   |
+--------------------+
           |
           v
+--------------------+     +--------------------+     +--------------------+
|  Transaction       |<--->|  Blockchain        |     |  Bank Account      |
|  Service           |     |  Clients           |     |  Clients           |
+--------------------+     +--------------------+     +--------------------+
           |
           v
+--------------------+     +--------------------+
|  Firefly III       |<--->|  Database          |
|  Client            |     |  (Pocketbase)      |
+--------------------+     +--------------------+
```

---

## 4. Components

### 4.1 Configuration

- **Description**: Stores application settings, including wallet addresses, bank account details, Firefly III API token, and account mappings.
- **Implementation**: Parsed from a JSON file into a Go struct at runtime.
- **Example**:

  ```go
  type Config struct {
      Firefly      FireflyConfig
      Wallets      map[string]string // e.g., "ethereum": "0xAddress"
      BankAccounts []BankAccountConfig
      Interval     string // e.g., "15m" for background task frequency
  }

  type BankAccountConfig struct {
      Name        string
      Provider    string // e.g., "enable_banking"
      Credentials map[string]string // Loaded from env vars (e.g., ENABLE_CLIENT_ID)
      Currencies  map[string]string // Currency to Firefly account ID
      Limit       int    // Default transaction limit (e.g., 10)
      FromDate    string // Optional start date (e.g., "2023-01-01")
      ToDate      string // Optional end date (e.g., "2023-12-31")
  }
  ```

### 4.2 Database

- **Description**: Tracks imported transactions to prevent duplicates.
- **Implementation**: SQLite with a table (`imported_transactions`) storing transaction IDs and metadata, persisted via Docker volume.

### 4.3 Blockchain Clients

- **Description**: Retrieve transaction data from cryptocurrency wallets.
- **Implementation**: Modular clients implementing the `BlockchainClient` interface:
  - `EthereumClient`: Uses Etherscan API.
  - `SolanaClient`: Uses Solscan API.
  - `SUIClient`: Uses SUI's JSON-RPC API.

### 4.4 Bank Account Clients

- **Description**: Retrieve balance and transaction data from bank accounts via Enable Banking API.
- **Implementation**: Modular clients implementing the `BankAccountClient` interface:
  - `EnableBankingClient`: Supports multiple banks (e.g., Revolut as ASPSP).
- **Key Functions**:
  - `FetchBalances()`: Retrieves account balances.
  - `FetchTransactions(limit int, fromDate, toDate string)`: Retrieves transactions with customization.

### 4.5 Firefly III Client

- **Description**: Communicates with Firefly III's API to manage accounts and transactions.
- **Implementation**: Injectable client with methods like `CreateTransaction` and `GetCurrencyID`.

### 4.6 Services

- **Description**: Encapsulate business logic for processing transactions from both sources.
- **Implementation**: `TransactionService` handles fetching, parsing, and importing.

### 4.7 Service Manager

- **Description**: Coordinates concurrent operations across services and manages background tasks.
- **Implementation**: Uses Go contexts and channels to manage goroutines, with signal handling for lifecycle management.
- **Background Mode**: Runs indefinitely, periodically fetching data at a configurable interval (e.g., every 15 minutes).
- **Foreground Mode**: Runs interactively, with the CLI controlling execution.

### 4.8 CLI

- **Description**: Command-line interface supporting foreground and detached modes.
- **Implementation**:
  - **Foreground Mode**: Runs with `--foreground` flag, blocks until interrupted (e.g., Ctrl+C).
  - **Detached Mode**: Runs with `--detach` flag, operates as a background service.
- **Signal Handling**: Listens for OS signals (e.g., SIGTERM, SIGINT) to trigger graceful shutdown.

---

## 5. Interfaces

### 5.1 BlockchainClient Interface

```go
type BlockchainClient interface {
    FetchTransactions(address string) ([]Transaction, error)
}
```

### 5.2 BankAccountClient Interface

```go
type BankAccountClient interface {
    FetchBalances() ([]Balance, error)
    FetchTransactions(limit int, fromDate, toDate string) ([]Transaction, error)
}
```

### 5.3 FireflyClient Interface

```go
type FireflyClient interface {
    CreateTransaction(accountID, currencyID string, t Transaction) error
    GetCurrencyID(accountID string) (string, error)
}
```

### 5.4 Database Interface

```go
type Database interface {
    IsTransactionImported(txID string) (bool, error)
    MarkTransactionAsImported(txID string) error
}
```

---

## 6. Data Management

### 6.1 Data Flow

1. Load configuration from TOML file.
2. Fetch transaction data from blockchain APIs or Enable Banking API.
3. Query SQLite to filter out already imported transactions.
4. Parse and map transactions to Firefly III's format.
5. Submit transactions to Firefly III via API.
6. Update SQLite with imported transaction records.

### 6.2 Data Structures

- **Transaction**:
  ```go
  type Transaction struct {
      ID          string
      Currency    string
      Amount      float64
      Type        string // "deposit" or "withdrawal"
      Description string
      Timestamp   time.Time
  }
  ```
- **Balance**:
  ```go
  type Balance struct {
      Currency string
      Amount   float64
  }
  ```

### 6.3 Persistence

- **SQLite Database via Pocketbase**: Stored in a Docker volume (e.g., `/app/data/importer.db`) to persist imported transaction records across container restarts.
- **Configuration**: TOML file stored in the same volume or passed via environment variables.

---

## 7. Error Handling and Logging

### 7.1 Error Handling

 Use Pocketbase to handle errors and exceptions, ensuring that all errors are logged and reported to the user. Implement retry logic for transient errors (e.g., network issues) with exponential backoff.
- **Transient Errors**: Retry with exponential backoff (e.g., 1s, 2s, 4s).
- **Permanent Errors**: Log and notify the user, e.g., invalid credentials or API limits reached.
- **Graceful Shutdown**: Handle OS signals (e.g., SIGTERM, SIGINT) to stop background tasks and close resources.

---

## 8. Security Considerations

### 8.1 API Keys and Tokens

- Load sensitive credentials (e.g., `client_id`, `client_secret`) from environment variables.
- Store access/refresh tokens in SQLite, optionally encrypted.

### 8.2 Data Privacy

- Avoid logging raw API responses with sensitive data.
- Clear temporary data from memory after processing.

---

## 9. Testing Strategy

### 9.1 Unit Tests

- Test components (e.g., clients, services) with mocked dependencies.

### 9.2 Integration Tests

- Verify interactions with external APIs using test environments or mocks.

### 9.3 End-to-End Tests

- Simulate full workflows with sample data to validate behavior.
- Test signal handling and shutdown scenarios.

---

## 10. Deployment and Maintenance

### 10.1 Deployment

- **Docker Image**: Multi-stage Dockerfile builds and packages the Go binary into a lightweight Alpine image.
- **Docker Compose**: Orchestrates the application and dependencies (e.g., Firefly III instance).
- **Volumes**: Persist SQLite database and configuration files.
- **Environment Variables**: Pass sensitive credentials and configuration settings.

### 10.2 Maintenance

- Monitor API usage to respect rate limits.
- Update dependencies and adapt to API changes as needed.
- Use Docker logs for monitoring and troubleshooting.

---

## 11. Conclusion

This EDD provides a robust, modular, and scalable design for importing cryptocurrency wallet and bank account transactions into Firefly III, enhanced to run as a background service within a Docker container. The architecture ensures maintainability and extensibility, while Docker and Docker Compose offer portability and ease of deployment. The design supports both foreground and detached modes, providing flexibility for various use cases.

---

## 12. Appendices

### 12.1 References

- [Firefly III API Documentation](https://api-docs.firefly-iii.org/)
- [Enable Banking API Documentation](https://enablebanking.com/docs)
- [Etherscan API](https://etherscan.io/apis)
- [Solscan API](https://docs.solscan.io/)
- [SUI API Reference](https://docs.sui.io/sui-api-ref)
- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)

### 12.2 Glossary

- **Hexagonal Architecture**: Isolates business logic using ports and adapters.
- **Inversion of Control (IoC)**: Delegates object creation to a framework or container.
- **Docker**: Platform for developing, shipping, and running applications in containers.
- **Docker Compose**: Tool for defining and running multi-container Docker applications.
- **SOLID Principles**: Set of design principles for object-oriented programming.
- **TOML**: Configuration file format (Tom's Obvious, Minimal Language).
- **SQLite**: Lightweight, serverless SQL database engine.
- **Pocketbase**: Lightweight backend for managing data and authentication.
- **API Rate Limiting**: Restriction on the number of API requests in a given time frame.
- **Concurrency**: Ability to run multiple tasks simultaneously.
- **Goroutine**: Lightweight thread managed by the Go runtime.
- **Signal Handling**: Mechanism to intercept and respond to OS signals (e.g., SIGTERM, SIGINT).
- **Exponential Backoff**: Algorithm for retrying operations with increasing wait times.
- **Docker Volume**: Persistent storage for Docker containers.
- **Environment Variables**: Variables set in the operating system to configure applications.
- **Open API Specification (OAS)**: Standard for defining RESTful APIs.
- **Account Servicing Payment Service Provider (ASPSP)**: Financial institution providing account services.
- **Account Information Service Provider (AISP)**: Third-party service provider accessing account information.
- **Payment Initiation Service Provider (PISP)**: Third-party service provider initiating payments.
- **Authorization Code Flow**: OAuth 2.0 flow for obtaining access tokens.
- **Access Token**: Token used to authenticate API requests.
- **Refresh Token**: Token used to obtain a new access token.
- **Bearer Token**: Token used in HTTP headers for authentication.
- **OAuth 2.0**: Authorization framework for third-party applications.
- **API Key**: Unique identifier for authenticating API requests.
- **API Secret**: Secret key used to sign API requests.
- **API Endpoint**: URL for accessing a specific API resource.
- **API Rate Limit**: Maximum number of API requests allowed in a given time frame.
- **API Throttling**: Mechanism to control the rate of API requests.
- **API Pagination**: Technique for splitting large API responses into smaller pages.
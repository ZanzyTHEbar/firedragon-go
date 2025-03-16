### 1. Project Overview and Goals

- **Purpose**: Explain that the application integrates cryptocurrency wallets (e.g., Ethereum, Solana, SUI) and bank accounts (via APIs like Enable Banking) with Firefly III to automate transaction tracking.
- **Scope**: Define what's included—supported blockchains, bank account providers (e.g., Revolut), and Firefly III syncing—and what's excluded (e.g., unsupported currencies).
- **Goals**: Highlight key objectives, such as automating imports, ensuring data accuracy, and maintaining security.
- **Constraints and Assumptions**: Note limitations like API rate limits or assumptions like the user having a running Firefly III instance.

---

### 2. System Architecture

- **High-Level Design**: Describe the use of a **hexagonal architecture** (ports and adapters pattern) to separate business logic from external systems (e.g., APIs, databases).
- **Key Components**:
  - **Blockchain Clients**: Fetch and parse transaction data from crypto wallets.
  - **Bank Account Clients**: Retrieve balances and transactions from banks.
  - **Firefly III Client**: Push data to Firefly III's API.
  - **Services**: Process transactions, prevent duplicates, and map data.
  - **Service Manager**: Coordinate background tasks and concurrency.
- **Design Principles**: Explain how **dependency injection (DI)** and **inversion of control (IoC)** are used to keep components loosely coupled.

---

### 3. Component Details

For each component, provide:

- **Responsibilities**: What it does (e.g., `BlockchainClient` fetches wallet transactions).
- **Interfaces**: Key methods it exposes (e.g., `FetchTransactions(address string) ([]Transaction, error)`).
- **Interactions**: How it connects to other components (e.g., `TransactionService` uses `BlockchainClient` and a database).
- **External Dependencies**: APIs it uses (e.g., Etherscan for Ethereum, Enable Banking for banks) and authentication methods (e.g., API keys, OAuth 2.0).

**Example**:

- `BankAccountClient`:
  - **Responsibility**: Fetch bank transactions and balances.
  - **Interface**: `GetTransactions(accountID string) ([]Transaction, error)`.
  - **Interaction**: Called by `TransactionService`, stores data in SQLite.
  - **API**: Enable Banking, authenticated via OAuth 2.0.

---

### 4. Data Flow and Processing

- **Workflow**:
  1. Load configuration from files or environment variables.
  2. Fetch transaction data from wallets or bank accounts.
  3. Check SQLite for duplicates.
  4. Transform data into Firefly III's format.
  5. Send transactions to Firefly III API.
  6. Log successful imports in SQLite.
- **Data Structures**: Define structs like:
  ```go
  type Transaction struct {
      ID          string
      Date        time.Time
      Amount      float64
      Currency    string
      Description string
  }
  ```
  - Explain how fields map to Firefly III's API (e.g., `Amount` to `amount`, `Date` to `date`).
- **Transformation**: Detail how raw API data (e.g., Etherscan JSON) is parsed and standardized.

---

### 5. Concurrency and Service Management

- **Concurrency**: Describe how **goroutines** handle simultaneous fetching from multiple sources (e.g., one per wallet or account).
- **Service Manager**: Explain its role in:
  - Starting and stopping services.
  - Running periodic tasks in detached mode (e.g., every 15 minutes).
- **Lifecycle**: Detail startup, task scheduling, and shutdown using `context` for cancellation and signal handling (e.g., SIGTERM) for graceful exits.

---

### 6. Security Considerations

- **Credentials**:
  - Store API keys and tokens in **environment variables** (e.g., `ETHERSCAN_API_KEY`).
  - Use SQLite to persist OAuth tokens (e.g., Enable Banking access/refresh tokens), optionally encrypted.
- **Data Privacy**: Avoid logging sensitive data like raw transactions.
- **Authentication**: Explain OAuth 2.0 flows for bank APIs, including token refresh logic.

---

### 7. Deployment and Operation

- **Docker**:
  - Use a multi-stage Dockerfile to build and run the Go application.
  - Use Docker Compose to orchestrate with dependencies (e.g., Firefly III).
- **Run Modes**:
  - **Foreground**: Interactive CLI for testing.
  - **Detached**: Background service for continuous operation.
- **Signal Handling**: Graceful shutdown on OS signals, ensuring pending tasks complete.

---

### 8. Diagrams and Visual Aids

- **Flow Chart**: Outline the main workflow (startup → fetch → process → import → shutdown).
- **Transaction Flow**: Visualize fetching, filtering, mapping, and importing steps.
- **UML**:
  - **Class Diagram**: Show component relationships (e.g., `TransactionService` depends on `BlockchainClient`).
  - **Sequence Diagram**: Illustrate a transaction import sequence in detached mode.

---

### 9. Configuration and Environment

- **Config File**: Define `config.json` structure:
  ```json
  {
    "wallets": [{ "chain": "ethereum", "address": "0x..." }],
    "banks": [{ "provider": "revolut", "account_id": "123" }],
    "firefly": { "url": "http://localhost:8080", "token": "xyz" },
    "interval": "15m"
  }
  ```
- **Environment Variables**: List essentials like:
  - `ENABLE_CLIENT_ID`: Enable Banking OAuth client ID.
  - `FIREFLY_API_TOKEN`: Firefly III API token.

---

### 10. Testing and Validation

- **Testing**:
  - **Unit Tests**: Test individual components (e.g., parsing logic in `BlockchainClient`).
  - **Integration Tests**: Mock API calls to validate interactions.
  - **End-to-End Tests**: Simulate imports with sample data.
- **Validation**:
  - Check data accuracy by comparing imported transactions with source data.
  - Measure performance (e.g., time to process 100 transactions).

---

### 11. External Resources

- **APIs**:
  - [Firefly III API](https://api-docs.firefly-iii.org/)
  - [Enable Banking API](https://enablebanking.com/docs)
  - [Enable Banking API Reference](https://enablebanking.com/docs/api/reference)
  - [Etherscan API](https://etherscan.io/apis)
- **Libraries**: Mention Go packages like `net/http` (API calls), `database/sql` (SQLite).
- **References**: Link to relevant GitHub repos or docs for tools used.

---

### 12. Glossary

- **Hexagonal Architecture**: A design pattern separating core logic from external systems.
- **Goroutines**: Lightweight threads in Go for concurrency.
- **OAuth 2.0**: Authentication protocol for bank APIs.

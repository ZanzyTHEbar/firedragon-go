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

---

# FireDragon Project Rules

ALWAYS UPDATE MEMORY BANK AS YOU WORK. 

ALWAYS CHECK MEMORY BANK STATUS BEFORE STARTING A NEW TASK.

## Project Patterns

### Project-Specific Rules

The global project module is `github.com/ZanzyTHEbar/firedragon-go`, and the project is called `firedragon-go`. The project is a multi-currency transaction processing system. The project is designed to be modular and extensible, with a focus on clean architecture and separation of concerns. The project is built using Go 1.23+ and follows the Go community's best practices for code organization, testing, and documentation.

### Code Organization
1. All new code should follow the hexagonal architecture pattern
2. Interfaces should be defined in the interfaces/ directory
3. Implementations should be in the adapters/ directory
4. Core business logic belongs in the services/ directory
5. Common utilities go in pkg/
6. Internal shared code goes in internal/

### Naming Conventions
1. Use descriptive, full words for names
2. Interface names should describe behavior (e.g., TransactionProcessor)
3. Implementation names should include context (e.g., EnableBankingClient)
4. Test files should end with _test.go
5. Mock files should end with _mock.go

### Error Handling
1. Always use wrapped errors with context
2. Log errors at the appropriate level
3. Include relevant transaction/operation IDs in errors
4. Use custom error types for specific scenarios
5. All Errors in the project MUST use the `github.com/ZanzyTHEbar/errbuilder-go` package.
6. Usage of this library can already be found in the `firefly/errors.go` file.

### Configuration
1. Use TOML format for configuration files
2. Support environment variable overrides
3. Place configs in ~/.config/firedragon/
4. Use viper for configuration management

### Development Workflow
1. Use make commands for common tasks
2. Run make dev for development
3. Use make test for testing
4. Clean with make clean

### Testing
1. Write unit tests for all new code
2. Include integration tests for adapters
3. Use table-driven tests where appropriate
4. Mock external dependencies

## Project Intelligence

### Critical Paths
1. Transaction processing pipeline
2. External API integrations
3. Database operations
4. Error handling and recovery

### Known Challenges
1. Rate limiting across multiple APIs
2. Transaction deduplication logic
3. Multi-currency handling
4. OAuth2 token management

### User Preferences
1. Configuration via TOML files
2. Command-line interface primary interaction
3. Minimal manual intervention needed
4. Clear error messages and logging

### Tool Usage
1. make for build and development tasks
2. air for live reload during development
3. NATS for message processing
4. SQLite for local storage

### Evolution Notes
1. Started with basic CLI structure
2. Added NATS integration for reliability
3. Implementing adapters progressively
4. Building monitoring system incrementally

## Project-Specific Rules

### Code Style
1. Use gofmt for formatting
2. Follow Go best practices
3. Document all exported items
4. Keep functions focused and small

### Architecture Rules
1. Maintain clean interface boundaries
2. Use dependency injection
3. Keep core domain pure
4. Abstract external dependencies

### Security Rules
1. Never commit credentials
2. Use environment variables for secrets
3. Validate all external input
4. Use TLS for all API connections

### Performance Rules
1. Implement rate limiting
2. Use connection pooling
3. Cache frequently accessed data
4. Batch operations where possible 

### Pocketbase Rules

We are using pocketbase 0.26.x. There is no `models` package in this version, everything has been refactored into `core` package, and various sub-packages. There is no `schemas` package either. 

Collections are Tables and Records are Rows in traditional parlance.

1. Use indexes for frequently queried fields
2. Use transactions for batch operations
3. Monitor performance with profiling tools
4. Optimize queries for speed
5. Use pagination for large result sets
6. Avoid N+1 query problems
7. Use prepared statements for repeated queries
8. Use caching for frequently accessed data
9. Use connection pooling for database connections
10. Use async processing for long-running tasks
11. Use background jobs for non-urgent tasks
12. Use a CDN for static assets
13. Use gzip compression for API responses
14. Use HTTP/2 for API connections

---

---
description: Global Tracking of Project Organization
globs: 
---
# Architecture Reorganization Plan

## Current Status

The project currently has a basic hexagonal architecture but needs reorganization to better support both FireflyClient and PocketBase features. The current structure includes:

- `interfaces/` for domain contracts
- `adapters/` for implementations
- `internal/` for core utilities
- `firefly/` at root level (needs to be moved)
- Various PocketBase directories that need better organization

## Proposed Directory Structure

```
firedragon-go/
├── adapters/
│   ├── banking/          # Banking integrations
│   ├── blockchain/       # Blockchain integrations
│   ├── firefly/          # Moved from root
│   └── pocketbase/       # New PB implementations
├── interfaces/
│   ├── clients.go
│   ├── event.go
│   └── pocketbase/       # New PB interfaces
├── services/            # New directory for business logic
│   ├── auth/
│   ├── sync/
│   ├── transaction/
│   └── user/
├── internal/            # Slimmed down to core utilities
│   ├── config/
│   ├── logger/
│   └── utils/
├── pb_migrations/       # Organized by feature
│   ├── auth/
│   ├── users/
│   └── transactions/
├── pb_hooks/           # Organized by feature
│   ├── auth/
│   ├── users/
│   └── transactions/
└── main.go
```

## Interface Definitions

### PocketBase Core Interfaces

```go
// interfaces/pocketbase/app.go
type PocketBaseApp interface {
    Initialize() error
    Serve() error
    Shutdown() error
}

// interfaces/pocketbase/auth.go
type AuthManager interface {
    Authenticate(ctx context.Context, credentials auth.Credentials) (*auth.Session, error)
    ValidateSession(ctx context.Context, token string) bool
    RevokeSession(ctx context.Context, token string) error
}

// interfaces/pocketbase/data.go
type DataManager interface {
    CreateRecord(ctx context.Context, collection string, data interface{}) error
    GetRecord(ctx context.Context, collection, id string) (interface{}, error)
    UpdateRecord(ctx context.Context, collection, id string, data interface{}) error
    DeleteRecord(ctx context.Context, collection, id string) error
}
```

## PocketBase Features

### 1. Authentication System
- OAuth2 integration
- Session management
- Role-based access control
- API key management
- Token refresh handling
- Multi-factor authentication support

### 2. Data Models
- User profiles and preferences
- Account settings and configurations
- Transaction records and history
- Budget tracking and goals
- Category management
- Tags and metadata
- Audit logs

### 3. Migrations
- User schema and authentication
- Transaction records and metadata
- Settings and preferences
- Security and permissions
- Integration mappings
- Audit trail schema

### 4. Hooks
- Authentication events
- Data validation
- Sync triggers
- Audit logging
- Error handling
- Notification dispatch

## Integration Layer

### Service Layer Interfaces

```go
// services/sync/service.go
type SyncService interface {
    SyncTransactions(ctx context.Context) error
    SyncAccounts(ctx context.Context) error
    SyncCategories(ctx context.Context) error
    HandleWebhook(ctx context.Context, event WebhookEvent) error
}

// services/auth/service.go
type AuthService interface {
    Login(ctx context.Context, credentials auth.Credentials) (*auth.Session, error)
    ValidateToken(ctx context.Context, token string) bool
    RefreshToken(ctx context.Context, token string) (*auth.Session, error)
}

// services/event/service.go
type EventService interface {
    PublishEvent(ctx context.Context, event Event) error
    SubscribeToEvents(ctx context.Context, eventType string) (<-chan Event, error)
    UnsubscribeFromEvents(ctx context.Context, subscription string) error
}
```

## Implementation Plan

### Phase 1: Project Reorganization
1. Move FireflyClient to `adapters/firefly/`
2. Create new directory structure
3. Update import paths
4. Update build configuration
5. Update documentation

### Phase 2: PocketBase Setup
1. Define core interfaces
2. Implement basic PocketBase app
3. Set up authentication system
4. Create initial migrations
5. Configure basic hooks

### Phase 3: Integration Layer
1. Implement service layer
2. Set up event system
3. Create data mappers
4. Implement sync logic
5. Add error handling

### Phase 4: Feature Implementation
1. User management
2. Transaction synchronization
3. Settings management
4. Webhook handling
5. Audit logging

## Security Considerations

1. **Authentication**
   - Secure token handling
   - Session management
   - API key security
   - OAuth2 implementation

2. **Data Protection**
   - Encryption at rest
   - Secure communication
   - Sensitive data handling
   - Access control

3. **Audit & Compliance**
   - Action logging
   - Change tracking
   - Compliance reporting
   - Security scanning

## Testing Strategy

1. **Unit Tests**
   - Interface implementations
   - Service layer logic
   - Data mappers
   - Utility functions

2. **Integration Tests**
   - PocketBase integration
   - FireflyClient integration
   - Authentication flow
   - Data synchronization

3. **End-to-End Tests**
   - Complete workflows
   - Error scenarios
   - Performance testing
   - Security testing

## Documentation Requirements

1. **Architecture Documentation**
   - System overview
   - Component interaction
   - Data flow diagrams
   - Security model

2. **API Documentation**
   - Interface definitions
   - Endpoint descriptions
   - Authentication flows
   - Error handling

3. **Operational Documentation**
   - Setup guides
   - Configuration
   - Troubleshooting
   - Maintenance

## Monitoring and Observability

1. **Logging**
   - Structured logging
   - Log levels
   - Context tracking
   - Error reporting

2. **Metrics**
   - Performance metrics
   - Resource usage
   - Error rates
   - API latency

3. **Alerting**
   - Error thresholds
   - Resource limits
   - Security events
   - System health

## Next Steps

1. Begin project reorganization
   - Move directories
   - Update imports
   - Verify build
   - Update tests

2. Start PocketBase integration
   - Define interfaces
   - Create migrations
   - Implement hooks
   - Set up auth

3. Implement service layer
   - Create services
   - Add event system
   - Implement sync
   - Add security

4. Add features
   - User management
   - Transaction sync
   - Settings
   - Webhooks 

---

---
description: Global Project Rules
globs: 
---
# FireDragon Project Rules

ALWAYS UPDATE MEMORY BANK AS YOU WORK. 

ALWAYS CHECK MEMORY BANK STATUS BEFORE STARTING A NEW TASK.

## Project Patterns

### Project-Specific Rules

The global project module is `github.com/ZanzyTHEbar/firedragon-go`, and the project is called `firedragon-go`. The project is a multi-currency transaction processing system. The project is designed to be modular and extensible, with a focus on clean architecture and separation of concerns. The project is built using Go 1.23+ and follows the Go community's best practices for code organization, testing, and documentation.

### Code Organization
1. All new code should follow the hexagonal architecture pattern
2. Interfaces should be defined in the interfaces/ directory
3. Implementations should be in the adapters/ directory
4. Core business logic belongs in the services/ directory
5. Common utilities go in pkg/
6. Internal shared code goes in internal/

### Naming Conventions
1. Use descriptive, full words for names
2. Interface names should describe behavior (e.g., TransactionProcessor)
3. Implementation names should include context (e.g., EnableBankingClient)
4. Test files should end with _test.go
5. Mock files should end with _mock.go

### Error Handling
1. Always use wrapped errors with context
2. Log errors at the appropriate level
3. Include relevant transaction/operation IDs in errors
4. Use custom error types for specific scenarios
5. All Errors in the project MUST use the `github.com/ZanzyTHEbar/errbuilder-go` package.
6. Usage of this library can already be found in the `firefly/errors.go` file.

### Configuration
1. Use TOML format for configuration files
2. Support environment variable overrides
3. Place configs in ~/.config/firedragon/
4. Use viper for configuration management

### Development Workflow
1. Use make commands for common tasks
2. Run make dev for development
3. Use make test for testing
4. Clean with make clean

### Testing
1. Write unit tests for all new code
2. Include integration tests for adapters
3. Use table-driven tests where appropriate
4. Mock external dependencies

## Project Intelligence

### Critical Paths
1. Transaction processing pipeline
2. External API integrations
3. Database operations
4. Error handling and recovery

### Known Challenges
1. Rate limiting across multiple APIs
2. Transaction deduplication logic
3. Multi-currency handling
4. OAuth2 token management

### User Preferences
1. Configuration via TOML files
2. Command-line interface primary interaction
3. Minimal manual intervention needed
4. Clear error messages and logging

### Tool Usage
1. make for build and development tasks
2. air for live reload during development
3. NATS for message processing
4. SQLite for local storage

### Evolution Notes
1. Started with basic CLI structure
2. Added NATS integration for reliability
3. Implementing adapters progressively
4. Building monitoring system incrementally

## Project-Specific Rules

### Code Style
1. Use gofmt for formatting
2. Follow Go best practices
3. Document all exported items
4. Keep functions focused and small

### Architecture Rules
1. Maintain clean interface boundaries
2. Use dependency injection
3. Keep core domain pure
4. Abstract external dependencies

### Security Rules
1. Never commit credentials
2. Use environment variables for secrets
3. Validate all external input
4. Use TLS for all API connections

### Performance Rules
1. Implement rate limiting
2. Use connection pooling
3. Cache frequently accessed data
4. Batch operations where possible 

### Pocketbase Rules

We are using pocketbase 0.26.x. There is no `models` package in this version, everything has been refactored into `core` package, and various sub-packages. There is no `schemas` package either. 

Collections are Tables and Records are Rows in traditional parlance.

1. Use indexes for frequently queried fields
2. Use transactions for batch operations
3. Monitor performance with profiling tools
4. Optimize queries for speed
5. Use pagination for large result sets
6. Avoid N+1 query problems
7. Use prepared statements for repeated queries
8. Use caching for frequently accessed data
9. Use connection pooling for database connections
10. Use async processing for long-running tasks
11. Use background jobs for non-urgent tasks
12. Use a CDN for static assets
13. Use gzip compression for API responses
14. Use HTTP/2 for API connections

---

---
description: Rules for how to handle Placeholders
globs: 
---
# FireDragon Placeholder Rules

- All unimplemented placeholder comments MUST be prefixed with a `TODO:`
- Keep track of ALL placeholders & TODO sections, with a summary, in [placeholder.md] 
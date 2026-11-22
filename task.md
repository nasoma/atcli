# PROJECT: ATCLI — Africa's Talking Airtime Manager

## 1. Your role

You are a senior Go engineer specializing in command-line applications, terminal user interfaces, secure API integrations, and production-quality software architecture.

Your task is to **design, implement, test, and document a complete Go application named `atcli`** that integrates directly with the Africa's Talking API.

The application must support two interfaces:

1. **CLI:** Traditional commands for checking balances, sending airtime, managing configuration, and viewing transactions.
2. **TUI:** A polished interactive terminal application with keyboard navigation, forms, tables, confirmation dialogs, and transaction feedback.

Both interfaces must share the same underlying application services and API client.

Implement working software, not merely a proposal, pseudocode, or architecture document.

## 2. Project context

This application is primarily for personal use by a developer in Kenya who uses a MacBook Air with Apple Silicon.

It must also be easily installable by other Go developers through `go install`.

The application must:

- Be written entirely in Go.
- Work on macOS, with Linux compatibility where practical.
- Run as a standalone executable.
- Require no web frontend or separate backend server.
- Connect directly to Africa's Talking over HTTPS.
- Support sandbox and production environments.
- Default to Kenyan phone numbers and KES.
- Be secure, fast, lightweight, and easy to maintain.
- Avoid unnecessary abstractions and overengineering.

Use the latest stable compatible Go version and maintained dependencies, verified at implementation time.

## 3. Technology stack

Use:

| Component | Technology |
|---|---|
| Programming language | Go |
| CLI framework | Cobra |
| TUI framework | Bubble Tea |
| Terminal styling | Lip Gloss |
| TUI components | Bubbles |
| HTTP communication | Go standard `net/http` |
| JSON handling | Go standard `encoding/json` |
| Configuration | TOML or JSON |
| Credential storage | macOS Keychain, with secure alternatives |
| Transaction history | SQLite, preferably a pure-Go driver |
| Testing | Go standard testing framework |
| Releases | GitHub Actions |

Prefer minimal dependencies.

Use Go interfaces only where they improve testability or represent genuine component boundaries.

Do not use an ORM unless there is a compelling reason.

## 4. Application behaviour

The executable must be named:

`atcli`

Running the application without arguments:

```bash
atcli
```

Must launch the interactive TUI when attached to an interactive terminal.

Running commands with arguments must invoke the traditional CLI:

```bash
atcli balance
atcli airtime send +254712345678 100
```

Running without arguments in a non-interactive environment must produce helpful usage information rather than attempting to launch the TUI.

Additionally, support:

```bash
atcli tui
atcli --help
atcli --version
```

## 5. Africa's Talking API integration

Implement an API client using Go's HTTP standard library.

The API client must support:

- Application balance retrieval.
- Airtime distribution.
- Airtime transaction status lookup where supported.
- API authentication.
- Sandbox and production environments.
- Response validation.
- Structured errors.
- Request timeouts.

### API environments

Production:

`https://api.africastalking.com`

Sandbox:

`https://api.sandbox.africastalking.com`

### Airtime distribution

The documented airtime endpoint is:

`POST /version1/airtime/send`

Use the appropriate authentication headers and URL-encoded form data.

Recipients are supplied as a JSON-encoded array containing phone numbers and amounts.

Example recipient:

```json
{
  "phoneNumber": "+254712345678",
  "amount": "KES 100"
}
```

### Application balance

Implement application balance retrieval using the provider's documented user/application-data API.

Verify the exact endpoint, required query parameters, response structure, and authentication requirements from current documentation before implementation.

Parse the returned balance and currency into typed Go structures.

### Transaction status

Implement transaction status lookup using the provider's current documented status endpoint, if available.

Verify which identifiers are accepted, including whether the endpoint requires a request ID, transaction ID, or another identifier.

Do not invent unsupported API functionality.

### Important API requirements

Before implementing the HTTP client:

1. Consult current Africa's Talking documentation.
2. Confirm endpoints and HTTP methods.
3. Confirm authentication requirements.
4. Confirm request and response schemas.
5. Confirm sandbox-specific behaviour.
6. Confirm status lookup capabilities.
7. Document uncertainties or limitations.

Do not assume that an accepted airtime request means airtime was successfully delivered.

The immediate API response and final transaction status must be handled separately.

## 6. CLI commands

### 6.1 Account balance

```bash
atcli balance
```

Expected illustrative output:

```text
Africa's Talking Account

Environment: Production
Username:    myusername
Balance:     KES 2,450.00
```

Also support:

```bash
atcli balance --json
atcli balance --env sandbox
```

The JSON mode must print machine-readable JSON without decorative formatting or interactive prompts.

Balance retrieval must never trigger an airtime purchase.

### 6.2 Send airtime

Example:

```bash
atcli airtime send +254712345678 100
```

Expected interactive behaviour:

```text
Send Airtime

Recipient: +254712345678
Amount:    KES 100.00
Network:   Not verified
Account:   myusername
Mode:      Production

Proceed with airtime purchase? [y/N]:
```

Do not infer or display a mobile network unless it has been reliably verified.

After confirmation, perform the API request and display the resulting request ID and provider status.

Support these flags:

- `--currency`: Currency, default KES.
- `--env`: Sandbox or production.
- `--dry-run`: Validate and preview without submitting.
- `--json`: Output machine-readable results.
- `--yes`: Explicitly bypass the interactive confirmation, subject to safety limits.

Examples:

```bash
atcli airtime send +254712345678 100 --dry-run
atcli airtime send 0712345678 50
atcli airtime send +254712345678 100 --json
```

For `--yes`, require an explicit command-line amount and recipient. Do not permit ambiguous input or automatic production retries.

### 6.3 Transaction status

```bash
atcli airtime status ATQid_example
```

Display:

- Request or transaction ID.
- Recipient, when available.
- Amount.
- Current status.
- Last checked time.
- Provider message, when available.

Clearly distinguish:

- Pending or sent to provider.
- Successfully delivered.
- Failed.
- Unknown or unresolved.

### 6.4 Transaction history

```bash
atcli history
```

Display recent transactions in a readable table.

Support:

```bash
atcli history --limit 20
atcli history --status failed
atcli history --json
atcli history --env production
```

History must be based on locally persisted transaction records.

Do not claim the local history contains every transaction ever performed through the Africa's Talking dashboard.

### 6.5 Configuration

Support:

```bash
atcli config init
atcli config show
atcli config set default-currency KES
atcli config set environment sandbox
atcli config test
```

`config show` must never display API keys.

`config test` must verify connectivity and credentials through a non-monetary API operation.

### 6.6 Version and help

Support:

```bash
atcli version
atcli --version
atcli --help
atcli airtime --help
```

Use Cobra's built-in help and shell completion capabilities.

## 7. Interactive TUI requirements

Implement a polished terminal interface using Bubble Tea, Bubbles, and Lip Gloss.

The interface must be responsive to common terminal sizes and work correctly in macOS Terminal, iTerm2, Ghostty, and other modern terminal emulators.

### 7.1 Main dashboard

The dashboard should display:

```text
╭────────────────────────────────────────────╮
│ ATCLI — Africa's Talking                   │
│                                            │
│ Environment: Production                    │
│ Account: myusername                        │
│                                            │
│ Account Balance                            │
│ KES 2,450.00                               │
│                                            │
│  > Send Airtime                            │
│    Check Balance                           │
│    Transaction History                     │
│    Transaction Status                      │
│    Settings                                │
│    Exit                                    │
│                                            │
│ ↑↓ Navigate  Enter Select  q Quit         │
╰────────────────────────────────────────────╯
```

The balance and username in this example are illustrative.

Do not invent live account data.

### 7.2 Send airtime screen

Provide an interactive form with:

- Recipient phone number.
- Airtime amount.
- Currency.
- Active environment.
- Validation messages.
- Continue and cancel options.

The form must validate input before submission.

### 7.3 Confirmation screen

Before sending airtime, display a clear summary of the transaction.

Show:

- Exact recipient.
- Exact amount.
- Currency.
- Environment.
- Estimated remaining balance, only if reliable.

Require explicit confirmation.

For production purchases, make the action visually distinct from sandbox requests.

### 7.4 Transaction result screen

Display:

- Whether the provider accepted or rejected the request.
- Provider request ID.
- Transaction amount.
- Recipient.
- Initial status.
- Any provider error message.

Never interpret an initial `Sent` response as successful final delivery.

### 7.5 History screen

Display historical transactions in a navigable table.

Provide:

- Date and time.
- Recipient.
- Amount.
- Status.
- Environment.

Support selecting an entry to view its details.

### 7.6 Settings screen

Allow users to:

- View the active environment.
- Change the default environment.
- Change the default currency.
- Configure their Africa's Talking username.
- Update API credentials securely.
- Set transaction safety limits.
- Test API connectivity.

### 7.7 TUI usability

Implement:

- Arrow-key navigation.
- Enter to select.
- Escape to return.
- `q` to quit where appropriate.
- Loading indicators.
- Helpful errors.
- Responsive layouts.
- Graceful handling of terminal resize events.

Do not block the Bubble Tea event loop with network requests.

Use Bubble Tea commands for asynchronous network operations.

Prevent duplicate requests caused by repeated key presses while a purchase is pending.

## 8. Phone number handling

The application is initially focused on Kenya.

Accept Kenyan mobile numbers in these formats:

```text
0712345678
712345678
254712345678
+254712345678
```

Normalize them to E.164 format:

```text
+254712345678
```

Support applicable Kenyan mobile prefixes, including newer allocations, using a maintained validation approach.

Avoid inferring the mobile carrier from the prefix because of number portability.

Validate:

- Country code.
- Number structure.
- Allowed characters.
- Required length.
- Provider-supported destinations.

Keep the phone normalization logic independently testable.

Support international E.164 phone numbers where Africa's Talking permits airtime distribution, but validate supported currencies and countries.

## 9. Monetary amounts

Money handling must be precise.

Do not use floating-point arithmetic for monetary calculations.

Use integer minor units or a suitable decimal representation.

Examples:

```text
KES 10
KES 50
KES 100
KES 250.50
```

Validate amounts according to provider-supported limits.

The application must reject:

- Zero amounts.
- Negative amounts.
- Malformed amounts.
- Unsupported currencies.
- Values above the configured maximum.

Never silently round a transaction amount.

## 10. Security requirements

### Credentials

Store API keys securely.

Preferred implementation on macOS:

- macOS Keychain for sensitive API keys.
- Local configuration file for non-sensitive settings.

Support environment variables:

```bash
AT_USERNAME
AT_API_KEY
AT_ENVIRONMENT
```

Define credential precedence clearly.

Do not print credentials in logs, error messages, diagnostics, or JSON output.

Do not commit credentials to Git.

### Network security

- Require HTTPS.
- Validate TLS certificates.
- Use request timeouts.
- Limit response sizes.
- Avoid logging authentication headers.
- Handle unexpected HTTP responses.
- Prevent credentials from being sent to untrusted endpoints.

### Transaction safety

Implement:

1. Confirmation before monetary transactions.
2. Configurable maximum amount per transaction.
3. Optional daily spending limits based on locally known transactions.
4. Duplicate-submission protection.
5. Safe handling of network timeouts.
6. Explicit sandbox/production separation.
7. No automatic retries of potentially accepted purchases.
8. Transaction auditing.

### Duplicate prevention

Design the transaction flow to account for network failures occurring after the provider may have accepted a request.

Use a local durable operation ID and persist the purchase intent before network transmission.

If the provider supports an idempotency mechanism, use it according to documentation.

Otherwise, use a local deduplication strategy while clearly acknowledging that it cannot guarantee exactly-once delivery across all failure scenarios.

When a submission's result is uncertain, mark it unresolved and require reconciliation rather than immediately resending.

## 11. Local transaction history

Use SQLite for local persistence.

Store at least:

```text
id
local_operation_id
provider_request_id
recipient
amount
currency
environment
status
provider_message
created_at
updated_at
last_checked_at
```

Consider preserving separate submission and final-delivery states.

Use UTC timestamps internally and display local time to the user.

Transactions should have appropriate uniqueness constraints and indexes.

Use database migrations to maintain schema compatibility.

Never store API keys in the transaction database.

Ensure the application can recover safely after an unexpected termination during an airtime transaction.

## 12. First-run setup

When the application is first launched without configuration, guide the user through setup.

The setup flow must:

1. Welcome the user.
2. Explain sandbox and production modes.
3. Ask for an Africa's Talking username.
4. Securely collect the API key.
5. Store credentials securely.
6. Verify API connectivity without sending airtime.
7. Configure the default currency.
8. Configure transaction limits.
9. Open the main dashboard.

Use sandbox mode by default.

Do not switch to production implicitly.

If credential validation fails, explain the issue without revealing secrets.

## 13. Project architecture

Use a clean, maintainable Go project structure.

Suggested layout:

```text
atcli/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   ├── root.go
│   ├── balance.go
│   ├── airtime.go
│   ├── history.go
│   ├── config.go
│   └── version.go
├── internal/
│   ├── africastalking/
│   │   ├── client.go
│   │   ├── airtime.go
│   │   ├── balance.go
│   │   ├── status.go
│   │   └── models.go
│   ├── service/
│   │   ├── airtime.go
│   │   └── account.go
│   ├── tui/
│   │   ├── app.go
│   │   ├── dashboard.go
│   │   ├── send.go
│   │   ├── balance.go
│   │   ├── history.go
│   │   ├── settings.go
│   │   └── styles.go
│   ├── config/
│   │   ├── config.go
│   │   └── credentials.go
│   ├── storage/
│   │   ├── sqlite.go
│   │   └── migrations.go
│   ├── money/
│   │   └── money.go
│   └── phone/
│       └── phone.go
├── .github/
│   └── workflows/
│       └── ci.yml
├── .gitignore
├── README.md
└── LICENSE
```

Adjust the structure where appropriate, but retain clear separation between:

- HTTP API integration.
- Application/business logic.
- CLI interface.
- TUI interface.
- Configuration and secrets.
- Persistence.

Do not duplicate transaction logic across the CLI and TUI.

## 14. Go installation requirements

The application must support installation using:

```bash
go install github.com/USERNAME/atcli@latest
```

`USERNAME` is a configurable GitHub username placeholder, not a hardcoded repository identity.

Structure the Go module so the installation command works when published.

The project must also support:

```bash
git clone https://github.com/USERNAME/atcli.git
cd atcli
go mod tidy
go build -o atcli .
./atcli
```

Do not require Homebrew.

Do not require Docker.

Do not require a database server.

Prefer a pure-Go SQLite driver to avoid CGO requirements.

## 15. Testing requirements

Write meaningful automated tests.

### API client tests

Use `httptest.Server` and mocked API responses.

Cover:

- Successful balance retrieval.
- Authentication errors.
- Failed HTTP requests.
- Malformed JSON.
- Invalid response fields.
- Request timeouts.
- Airtime request acceptance.
- Airtime request rejection.
- Delayed transaction processing.
- Unknown transaction status.

### Validation tests

Cover:

- Kenyan phone number normalization.
- Invalid phone numbers.
- Currency parsing.
- Amount validation.
- Spending limits.
- Duplicate purchase detection.

### Persistence tests

Cover:

- Creating a pending transaction.
- Saving provider request IDs.
- Updating transaction status.
- Handling interrupted submissions.
- Reading transaction history.
- Database migrations.

### Interface tests

Test CLI argument handling and business logic used by the TUI.

Where practical, test TUI state transitions without a real terminal.

### Required checks

Run:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

Resolve failures before declaring the application complete.

## 16. Error handling and user experience

Use descriptive, actionable errors.

Examples:

```text
Error: Africa's Talking API credentials are missing.
Run: atcli config init
```

```text
Error: Invalid Kenyan phone number.
Expected format: +254712345678
```

```text
Error: Could not confirm the airtime request outcome.

The request may have reached the provider.
Do not repeat the purchase until its status is verified.
```

Distinguish among:

- Validation errors.
- Authentication errors.
- Network failures.
- Provider rejections.
- Insufficient balance.
- Uncertain transaction outcomes.

Use meaningful exit codes for CLI failures.

## 17. Documentation

Create a comprehensive `README.md` containing:

- Project overview.
- Feature list.
- Screenshots or text previews of the TUI.
- Prerequisites.
- Installation instructions.
- First-run setup.
- Africa's Talking credentials setup.
- Sandbox testing.
- CLI command examples.
- TUI keyboard shortcuts.
- Security considerations.
- Troubleshooting.
- Development commands.
- Running tests.
- Building for macOS and Linux.

Also document the limitation that local transaction history may not contain purchases made outside this application.

## 18. Development priorities

Implement the project in this order.

### Phase 1: Foundation

- Initialize the Go module.
- Configure Cobra.
- Implement configuration management.
- Implement secure credential storage.
- Add the Africa's Talking HTTP client.
- Implement phone and amount validation.

### Phase 2: Core CLI

- Account balance.
- Airtime sending.
- Transaction status.
- Sandbox support.
- Error handling.
- Confirmation flow.
- Unit tests.

### Phase 3: Transaction persistence

- Local SQLite database.
- Transaction history.
- Duplicate prevention.
- Uncertain transaction handling.
- Reconciliation.

### Phase 4: Interactive TUI

- Main dashboard.
- Balance view.
- Airtime form.
- Confirmation dialog.
- Transaction results.
- History table.
- Settings screen.

### Phase 5: Quality and distribution

- Complete test coverage of critical flows.
- Static analysis.
- README documentation.
- GitHub Actions CI.
- Go installation verification.
- macOS Apple Silicon build.
- Linux build verification where possible.

Do not stop after building the CLI. The project is incomplete until both CLI and TUI are working.

## 19. Explicit non-goals

For the initial release, do not build:

- A web frontend.
- A mobile application.
- A cloud-hosted backend.
- A user registration system.
- A multi-tenant SaaS platform.
- An authentication server.
- A payment collection platform.
- An MCP server.
- A Docker-based deployment requirement.

This is a local-first personal developer utility, not a commercial platform.

Bulk airtime distribution may be considered later, but it must not delay the core release.

## 20. Acceptance criteria

The project is complete when:

- [ ] `go install` works using the published module path.
- [ ] Running `atcli` opens the TUI in an interactive terminal.
- [ ] Running `atcli balance` retrieves the real account balance with valid credentials.
- [ ] A single airtime purchase can be submitted through the CLI.
- [ ] A single airtime purchase can be submitted through the TUI.
- [ ] Sandbox mode works.
- [ ] Production mode requires deliberate configuration.
- [ ] Phone number normalization works.
- [ ] Monetary amounts are handled precisely.
- [ ] API keys are securely stored and never exposed.
- [ ] Production purchases require explicit confirmation.
- [ ] Duplicate submissions are guarded against.
- [ ] Provider acceptance and final delivery are distinguished.
- [ ] Transaction history persists across application restarts.
- [ ] Network failures are handled safely.
- [ ] Automated tests pass.
- [ ] README contains accurate installation and usage instructions.

Never perform a real production airtime purchase as part of automated tests.

## 21. Final deliverables

Produce:

1. Complete Go source code.
2. Functional CLI.
3. Functional interactive TUI.
4. Africa's Talking API integration.
5. Local transaction storage.
6. Secure configuration management.
7. Automated tests.
8. GitHub Actions CI.
9. Detailed README.
10. Example usage and setup instructions.

Once implementation is complete:

- Run the test suite.
- Build the executable.
- Verify the CLI help commands.
- Verify TUI startup.
- Report any unverified functionality honestly.
- Explain how to configure real Africa's Talking credentials.
- Provide exact installation and usage commands.

If live credentials are unavailable, complete the application using realistic mocked responses and sandbox-compatible integration, then clearly identify what still requires live verification.

Do not claim to have successfully sent airtime unless an actual, explicitly authorized test has confirmed it.

## 22. Engineering expectations

Follow idiomatic Go practices.

Favor:

- Simple, readable code.
- Small focused functions.
- Explicit error handling.
- Minimal dependencies.
- Typed data structures.
- Context-aware network requests.
- Secure defaults.
- Testable business logic.
- Clear user feedback.

Avoid:

- Overengineering.
- Unnecessary interfaces.
- Magic constants.
- Hardcoded credentials.
- Global mutable application state.
- Blind HTTP retries.
- Unmaintained SDKs.
- Fake or unimplemented features presented as working.

Use up-to-date API and dependency documentation.

**Begin implementing the complete application now. Make reasonable engineering decisions independently, document assumptions, and deliver working code rather than stopping at planning.**
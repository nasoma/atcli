# ATCLI

A standalone Go CLI and terminal interface for Africa's Talking account balances,
single-recipient airtime purchases, status checks, and local SQLite history.
No backend, browser, Docker, Homebrew, or database server is needed.

## Build and install

Use Go 1.27.4 or later (Go can download the required toolchain automatically).
Install from the public `github.com/nasoma/atcli` repository:

```sh
go install github.com/nasoma/atcli@latest
```

Or clone and build locally:

```sh
git clone https://github.com/nasoma/atcli.git
cd atcli
go mod download
go build -o atcli .
go install .
./atcli --help
./atcli
```

`go install .` installs to GOBIN or `$(go env GOPATH)/bin`; add it to PATH.
Running without arguments opens the TUI only when stdin and stdout are terminals;
otherwise it prints help. `atcli tui` requires a terminal.

## Setup

Create an application and API key in the Africa's Talking dashboard. For sandbox,
use the username `sandbox` and its sandbox API key. Production uses your actual
application username and production API key. These are separate credential profiles.

Run `atcli config init`, or run `atcli` for the first-run form. The form defaults
to production and asks for environment, username, masked API key, currency, maximum
purchase amount and optional daily limit. Enter saves and tests credentials using
balance retrieval only, then opens the dashboard. Tab and arrow keys move between
fields. Enter your production application username and API key. Purchases spend
real funds and require confirmation. Sandbox remains an optional explicit mode.

API keys are stored using macOS Keychain or Linux Secret Service. There is no
plaintext credential fallback. Linux needs a working user D-Bus session and Secret
Service provider. If no credential store is available, supply environment variables
using your shell's secure secret-loading mechanism:

```sh
export AT_USERNAME=your_production_app_username
export AT_ENVIRONMENT=production
# Load AT_API_KEY from your secret manager; avoid putting it in shell history.
atcli config test
```

Precedence: `--env` > AT_ENVIRONMENT > saved environment;
AT_USERNAME > selected profile username; AT_API_KEY > selected profile's OS secret.
Environment credentials apply to the selected environment, so take care to load the
matching pair. Sandbox rejects usernames other than `sandbox`.
`config show` never retrieves or displays keys.

Settings and local history live under the OS user configuration directory, in
`atcli` (macOS: `~/Library/Application Support/atcli`; Linux: `$XDG_CONFIG_HOME/atcli`
or `~/.config/atcli`). `ATCLI_HOME` overrides this for isolated testing.
Configuration and SQLite files use mode 0600; the app-created directory uses 0700.
History contains personal phone numbers and is not encrypted; protect backups.

## Commands

```sh
atcli balance
atcli balance --json --env production
atcli config test
atcli config show
atcli config set environment production
atcli config set default-currency KES
atcli config set maximum 1000
atcli config set daily-limit 3000
atcli config set username myapp --env production
atcli airtime send 0712345678 100 --dry-run
atcli airtime send +254712345678 100 --env production
atcli airtime send +254712345678 100 --env production --yes --json
atcli airtime status ATQid_example --env production
atcli history --limit 20 --status failed --env production
atcli history --json
atcli completion zsh
atcli version
```

Amounts use integer minor units; there is no floating-point arithmetic or silent
rounding. Phone parsing uses maintained libphonenumber metadata; Kenyan local,
country-code and international formats are accepted, including 01 mobile ranges.
No carrier is inferred because numbers are portable.

Supported destination/currency pairs currently are Kenya/KES, Uganda/UGX,
Tanzania/TZS and Nigeria/NGN. Other destinations are not supported.
Provider-published bounds are enforced in addition to the
personal maximum (default 1000 in the selected currency). Account restrictions and
operator denominations can still cause rejection. Cross-country purchases need a
suitably provisioned app; Africa's Talking documents USD-base apps for cross-border
use. This tool's purchase currency must match the destination, not USD.
Daily limits are per account/environment/currency and UTC day; zero disables them.
They include accepted, unknown and interrupted intents, excluding known failures.
They only account for this local database, not dashboard or other-device purchases.

JSON purchase mode never prompts: use `--yes` to approve an explicit recipient and
amount. Dry runs need no credentials and never send requests. Errors go to stderr;
stdout remains JSON for results. Exit codes: 0 completed, 1 argument/configuration/local error, 3 API authentication
failure, 4 API failure or purchase rejection, 5 unresolved purchase outcome. For
purchases, inspect the JSON submission and status fields even on a nonzero exit.

## TUI

```text
╭─────────────────────────────────────────────────────────────╮
│ ATCLI — Africa's Talking  •  PRODUCTION                     │
│ Account: your-app                                           │
│                                                             │
│ › Send Airtime                                              │
│   Check Balance                                             │
│   Transaction History                                       │
│   Transaction Status                                        │
│   Settings                                                  │
│   Exit                                                      │
│                                                             │
│ ↑↓ Navigate  Enter Select  q Quit                           │
╰─────────────────────────────────────────────────────────────╯
```

Dashboard: arrows or j/k to navigate, Enter to select, q to quit.
Forms: Tab/arrows change field, Enter validates/continues; Escape returns.
Purchase confirmation: Enter submits exactly once, n edits, Escape cancels.
Production is marked in red. Network requests run asynchronously with a spinner;
purchase controls are locked while pending. Ctrl+C exits at any time, leaving the
durable intent available for reconciliation if a request was interrupted.
History: arrows select, Enter shows full details. Settings: Enter securely saves
and tests using a non-monetary balance operation. To update the key, enter a new
one; blank retains the saved key. Settings are never saved implicitly on exit.
Select Check Balance to retrieve your current account balance.
A terminal of at least 80×24 is recommended for tables and setup; resize events
update the form width and table height.

## Purchase safety and reconciliation

Before every network submission, a cryptographically random local operation ID
and intent are committed with SQLite synchronous FULL. The operation ID is sent
as `Idempotency-Key`, and `maxNumRetry=0` disables provider delivery retries.
The HTTP client never automatically retries purchases or follows redirects.
A serial SQLite reservation prevents concurrent processes from bypassing local
limits or deduplication. Identical recipient/amount/currency/account/environment
purchases are blocked while pending or unknown; delivered duplicates are blocked
for two minutes. Known failures may be retried by a new explicit purchase.

`Sent` means **accepted/pending**, never final delivery. Only a successful status
lookup maps to delivered. Unexpected responses, timeouts, process interruption and
missing IDs remain unknown. Do not repeat them: check the recorded provider ID
with `airtime status`, or reconcile in the provider dashboard if no ID was saved.
Unresolved purchases remain in history until their status can be confirmed. If the
provider cannot resolve a purchase, contact support before sending it again.
Local deduplication and provider idempotency cannot guarantee exactly-once delivery
across lost databases, devices, or undocumented provider retention windows.

History is local only, scoped by selected environment and labelled by account.
It does not import every transaction performed through the Africa's Talking dashboard.
Status lookup accepts local operation IDs or provider ATQid identifiers. A provider
ID absent from local history can be checked directly; recipient and amount are
then unavailable, and the lookup does not add a purchase to local history.

## API reference

Africa's Talking API endpoints and account restrictions:

- [Application SDK](https://github.com/AfricasTalkingLtd/africastalking-python/blob/master/africastalking/Application.py): GET `/version1/user?username=...`, `UserData.balance`.
- [Airtime SDK](https://github.com/AfricasTalkingLtd/africastalking-python/blob/master/africastalking/Airtime.py): POST `/version1/airtime/send`, URL-encoded username and JSON recipients, `Idempotency-Key`, `maxNumRetry`.
- [Official Postman airtime collection](https://www.postman.com/africastalking/africa-s-talking-apis/documentation/nu3hk0m/airtime): GET `/query/transaction/find?username=...&transactionId=ATQid_...`, response `status`. This accepts the provider ATQid identifier, not the local operation ID; local IDs are resolved through SQLite first.
- [Official request limits](https://help.africastalking.com/en/articles/1399779-what-are-the-transaction-limits-for-airtime): KE 10–10000, UG 50–200000, TZ 1000–200000, NG 50–5000 (local currency).
- [Cross-border restrictions](https://help.africastalking.com/en/articles/8300724-cross-country-request-not-permitted-why-am-i-getting-this-error).

Hosts are fixed to `https://api.africastalking.com` and
`https://api.sandbox.africastalking.com`. Requests send `apiKey` and
`Accept: application/json`, validate TLS, time out after 20 seconds and cap responses
at 1 MiB. There is no user-configurable endpoint that could receive your key.
Check final delivery status with `atcli airtime status`; ATCLI does not receive
callbacks. Sandbox simulates purchases without delivering real airtime. Production
requires valid credentials, sufficient balance and an account provisioned for the
recipient's market. Operator availability and account restrictions may affect delivery.

## Troubleshooting

- Missing credentials: run `config init`, or load the AT_* variables for this profile.
- Keychain denied: allow access in macOS Keychain; on Linux check Secret Service.
- Authentication failed: check app username, environment and API key pairing.
- Duplicate/unresolved: reconcile history before repeating; do not delete the DB.
- Provider rejection: check available balance, destination and account provisioning.
- No ID after a timeout: use the operation timestamp and recipient in the dashboard.
- TUI unavailable: use a real interactive terminal, or use CLI commands.
- Currency/amount refused: inspect configured maximum and supported market limits.

## Development and distribution

```sh
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go build ./...
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/atcli-darwin-arm64 .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/atcli-linux-amd64 .
```

Version tags publish macOS and Linux archives for arm64 and amd64, with checksums,
through GitHub Releases.

MIT licensed. See LICENSE.

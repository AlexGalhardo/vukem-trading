# Railway publication — configurations and handoff

Status: project provisioned; web, engine and PostgreSQL healthy on 2026-11-03; external integrations
await real credentials. See the destinations below and confirm health in the CLI.
Do not paste tokens or client secrets into the conversation or into versioned files.

## Canonical host: app.trading.vukem.dev

The terminal lives at <https://app.trading.vukem.dev>; the documentation stays at
<https://trading.vukem.dev> (the `site` service). The two are separate Railway services and
domains.

Environment variables, names only (set them in the Railway editor; never commit values):

| Variable | Service | Required value or source |
| --- | --- | --- |
| `APP_NAME` | web, engine | `vukem-trading` |
| `APP_ENV` | web, engine | `production` |
| `APP_DEBUG` | web, engine | `false` |
| `APP_KEY` | web, engine | Same generated key on both (`vukem key:generate`) |
| `APP_PREVIOUS_KEYS` | web, engine | Empty, or the former keys after a rotation |
| `APP_URL` | web, engine | `https://app.trading.vukem.dev` exactly: https, no path. The web refuses to boot in production with any other shape (`accounts.ValidateAppURL`) |
| `HOST` | web, engine | `0.0.0.0` |
| `PORT` | web | Provided by Railway; do not set it |
| `ENGINE_HEALTH_PORT` | engine | Empty, so the engine health check uses `PORT` |
| `DATABASE_URL` | web, engine | Reference to the PostgreSQL service, private network |
| `SESSION_DRIVER` | web, engine | `database` |
| `CACHE_STORE` | web, engine | `database` |
| `MAIL_MAILER` | web, engine | `log` |
| `OWNER_GOOGLE_EMAIL` | web | The owner's verified Google address |
| `SOCIAL_GOOGLE_CLIENT_ID` | web | Google Web OAuth client id |
| `SOCIAL_GOOGLE_CLIENT_SECRET` | web | Google Web OAuth client secret |
| `ALPACA_API_KEY`, `ALPACA_API_SECRET` | web (history import), engine | Alpaca Paper keys; optional |
| `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID` | engine, and web for the daily ranking | Optional operational alerts; the daily ranking job runs in the process with `SCHEDULE_IN_SERVE=true` (web by default), so that service needs them to send the ranking |
| `PAPER_TRADING`, `BINANCE_TESTNET_API_KEY`, `BINANCE_TESTNET_API_SECRET` | whichever service runs `reco:paper` | Optional; paper order placement stays off unless `PAPER_TRADING=true` |
| `MCP_TOKEN` | web | Optional; leave unset to keep `/mcp` off |
| `LIVE_TRADING_ENABLED`, `LIVE_TRADING_ACK`, `LIVE_MAX_ORDER_VALUE`, `LIVE_MAX_DAILY_LOSS_BPS` | none | Leave unset; there is no live adapter |

Google Cloud console, OAuth client of type Web application:

- Authorized JavaScript origin: `https://app.trading.vukem.dev`.
- Authorized redirect URI: `https://app.trading.vukem.dev/auth/google/callback` (the app builds it
  from `APP_URL`; `TestGoogleRedirectUsesTheCanonicalAppURL` pins the exact `redirect_uri`).
- Remove the old `scooby-web-production.up.railway.app` callback once the custom domain works.

Railway web service, in order:

1. Add the custom domain `app.trading.vukem.dev` to the `web` service and create the DNS records
   Railway prints (CNAME plus the verification TXT) in the `vukem.dev` zone. Wait for the verified
   state and a valid certificate.
2. Set the variables above, then deploy `main`. `railway.web.toml` runs `/app/app migrate` as the
   pre-deploy command (migrations finish before traffic moves; Vukem serializes them with an
   advisory lock) and gates traffic on `GET /ready` (120 second timeout, one replica).
3. Verify: `/` answers 302 to `/login`; `/login` 200; `/ready` 200; `/api/snapshot` with
   `Accept: application/json` and no session 401; `GET /auth/google/redirect` answers 302 to
   `accounts.google.com` with `redirect_uri=https://app.trading.vukem.dev/auth/google/callback`.
4. Sign in with the owner account, then deploy the engine and confirm the heartbeat on the
   dashboard.

Caching: hashed Vite assets under `/build/assets/` are served `public, max-age=31536000, immutable`
and gzip-compressed; every dynamic response (pages, `/api/*`, `/ready`) is `private, no-store`
(`TestPrivateResponsesAreNeverCached`), so no proxy or browser caches financial data.

## Provisioned destination

| Resource | ID/address |
| --- | --- |
| Project vukem-trading | `706b6cfc-45e2-4bfc-af6b-45a7c401ed37` |
| Environment production | `a0036a02-8b91-4463-96b5-622ed1fe3805` |
| web | `4611a2b2-b810-4941-9eb9-279a13bc7982` |
| worker (engine) | `aa2f67bb-69b5-470f-a676-abbfc9d7a806` |
| Postgres | `17f09ea5-508e-4e1e-9c3c-40eeed4f3600` |
| Web | <https://scooby-web-production.up.railway.app> |
| Google callback | <https://scooby-web-production.up.railway.app/auth/google/callback> |

Project: <https://railway.com/project/706b6cfc-45e2-4bfc-af6b-45a7c401ed37>

Web and engine connected to GitHub `AlexGalhardo/vukem-trading`, branch `main`.
Database with a private reference; application key generated and shared; values
checked without printing them. The owner's email and OAuth remain placeholders;
Alpaca and Telegram are empty/disabled. The user is creating Google/Telegram.
Initial region inherited from the provider: `us-east4-eqdc4a`, with no plan change.
Initial deploys confirmed SUCCESS at commit `4337761`: web
`266b5d5c-d668-4176-9bdc-7748ce9fee58`; engine
`80a0f3e2-9559-4269-a83b-facc46502e5c`. HTTPS smoke test: `/` 302; `/login` 200;
`/ready` 200; `/api/snapshot` without session 401. Engine healthcheck approved by the
provider; the readiness contract checks the heartbeat of both accounts in the database.

## Variable templates

- `.env.example`: development; optional credentials empty with placeholders
  identified in comments, avoiding external calls with fake keys.
- `.env.railway.web.example` and `.env.railway.engine.example`: templates per service.
- `.env.railway.web.local` and `.env.railway.engine.local`: private copies ignored
  by Git, prepared in the workspace. The application key is generated once and
  shared between the two; do not print these files.

The `.example` files are not loaded automatically by Railway. Apply
the variables in each service's editor or through the authenticated CLI. Do not run
`source` on the Railway templates: `${{Postgres.DATABASE_URL}}` is a provider reference, not a
shell expression. Adjust `Postgres` to the service's real name.
The domain, email and OAuth have explicit markers that must be replaced;
Alpaca and Telegram remain empty until real credentials are obtained. The Google
callback stays on the web domain, at `/auth/google/callback`.

Preparing local files does not prove the variables were applied on the provider;
record the project, services and domain only after confirmation from the Railway API.

## Non-secret information for the publication

Railway project/workspace and desired environment; service names/IDs; region and
accepted plan/budget; own domain or Railway domain; owner's Google email. If no
project exists, create a vukem-trading project and issue a token restricted to the
environment, instead of providing global access to the account.

Services: `web`, `worker` (the engine), `site`, PostgreSQL. Configuration via
`scripts/railway-configure.sh` and the current API; the TOMLs are historical references,
since the API rejected legacy Config as Code (ADR-0014). One replica for the engine,
Serverless/sleeping disabled. The worker keeps processing without the browser.
Use the private PostgreSQL URL between services. Only the web needs a
public domain; the engine's `/ready` works for healthcheck on the configured port.

## Secrets required, by destination

| Variable | Destination | How to obtain |
| --- | --- | --- |
| `RAILWAY_TOKEN` | Secure environment of Codex/CLI, not the app | Project/environment token; CLI OAuth session is also accepted |
| `APP_KEY` | web and engine | Generate once with `vukem key:generate`; share the same key and keep it |
| `DATABASE_URL` | web and engine | Reference to the project's PostgreSQL, preferably private network |
| `SOCIAL_GOOGLE_CLIENT_SECRET` | web only | OAuth credential of Web type in Google Cloud |
| `ALPACA_API_KEY`, `ALPACA_API_SECRET` | engine only | Keys for the Alpaca **Paper** account |
| `TELEGRAM_BOT_TOKEN` | engine, optional | Own bot for operational alerts |

The web currently also uses the Alpaca keys to import history at the owner's request.
To make this function available, configure the paper keys on the web
or separate the ingestion into a job in the future. Never expose these keys in `VITE_*`.

Additional settings:

- `APP_ENV=production`, `APP_DEBUG=false`, `APP_URL=https://DOMINIO-WEB`.
- `HOST=0.0.0.0`, `PORT` provided by Railway; `ENGINE_HEALTH_PORT` empty on
  Railway to use `PORT` (locally it may be `3001`).
- `OWNER_GOOGLE_EMAIL`: verified email admitted at bootstrap.
- `SOCIAL_GOOGLE_CLIENT_ID`: Web OAuth client ID.
- Exact Google callback: `https://DOMINIO-WEB/auth/google/callback`.
- `SESSION_DRIVER=database`, `CACHE_STORE=database`, `MAIL_MAILER=log`.
- `TELEGRAM_CHAT_ID`: owner's chat, if you choose alerts.
- CLI: `RAILWAY_PROJECT_ID`, `RAILWAY_ENVIRONMENT`, `RAILWAY_WEB_SERVICE`,
  `RAILWAY_ENGINE_SERVICE` identify the destination without ambiguity.

Configure OAuth consent and a test user if the Google app is in
Testing mode. A valid Google account other than the authorized one will be rejected.
The configuration email does not replace an already linked subject.

## Applying variables safely

Prefer the Railway variable editor or the Codex environment's secret bindings.
CLI 5.71.0 was installed in this workspace and verified by the published SHA256.
For automation, `railway variable set NAME --stdin --skip-deploys` reads a single
value from the input; use explicit project/environment/service selectors and do not
print the return value or files containing secrets. Avoid secret values in
shell arguments, screenshots and logs. Do not start an interactive login
on the assumption that a token must be displayed; use the configured authorization.

## Publication order

1. Provision PostgreSQL and backup. Configure variables and run
   `scripts/railway-configure.sh` with explicit IDs; do not define a TOML path.
2. Publish the web. The pre-deploy runs `/app/app migrate`; Vukem serializes PostgreSQL
   migrations with an advisory lock. Configure the pre-deploy timeout on the service.
3. Wait for a successful deploy and `/ready`. `--ci` alone only follows the build.
4. Publish the engine, wait for `/ready` and verify the dashboard heartbeat.
5. Check real Google login/callback/logout and restricted admission.
6. Import authorized B3 history and/or fetch IEX. Start with backtest.
7. Review policies and enable the strategy; accounts start **blocked**.
   Resuming PAPER is an explicit action in the panel. There is no live configuration.
8. Exercise cancellation and reconciliation in a dedicated paper account before using
   the personal paper account. Validate alerts if contracted/configured.

`scripts/railway-deploy.sh` publishes sequentially the services already configured and
waits for success status by deployment ID. Accepts a secure token or CLI OAuth
session. Does not create projects without information about the destination. The initial publication used
GitHub source connect and the equivalent API configuration, with no token in the chat.

## Backup and recovery

Native daily backup enabled and re-read via the API for the volume instance
`18f653a2-4123-4230-a47a-3d3a57d32ab6`. Returned schedule: `56 19 * * *`, retention
`518400` seconds (six days). The API does not offer custom retention in this contract;
the initial goal of seven days requires an external complement, if maintained. A confirmed schedule
does not mean the first backup or restore has already been validated.

`scripts/backup.sh` produces a custom dump, with restricted permissions. Set
`BACKUP_FILE` outside the repository and use encrypted private storage. Schedule an
external backup when needed to complement the provider's retention.
Initial operational target: RPO 24h, RTO 2h; the targets have not been measured in production.

Restore into an isolated database: `pg_restore --no-owner --no-acl` into a test database;
never point the engine/broker at the restore without blocking and reconciliation. The dump/restore of a test
PostgreSQL database was exercised, not production recovery.

Google recovery: an operator authenticated in the administrative environment may
run `/app/app owner:reset --confirm RESET-OWNER --reason "detailed reason"`.
This revokes sessions (session_version), records audit and reopens bootstrap with
`OWNER_GOOGLE_EMAIL`. It does not delete financial history and is not an HTTP route.
Keep the previous owner until possession of the new account is confirmed. No authentication bypass
is included in the production binary; `cmd/e2e` requires a build tag.

## What remains dependent on external evidence

Real Google login, Alpaca account permissions/data/fills, Railway availability,
real alerts, restoration on the provider, licensed B3 history and
observation of strategies over a significant period. Local tests do not prove these
results nor financial performance. P6 remains follow-up, not a box checked by synthetic test data.

The external ID of the Alpaca account is bound after the first reconciliation. Key
rotation must remain on the same paper account; pointing to another account blocks
the executor, without merging balances, orders and history. Account migration requires
its own administrative procedure.

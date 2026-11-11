# Set up locally with Docker

Run the whole stack (web, engine, PostgreSQL and a mail catcher) on your machine. Everything is paper trading; no real orders are possible.

## Requirements

- Docker with Compose v2.
- The Vukem CLI, to generate the application key: `go install github.com/AlexGalhardo/vukem/cmd/vukem@v0.41.0` (Go 1.27.4 or newer).
- Bun 1.4.4, only if you want to run the frontend tooling outside Docker.
- A Google Web OAuth client, to sign in (see [Google sign-in](#google-sign-in)).

## Steps

1. Clone the repository and enable the git hooks:

   ```sh
   git clone https://github.com/AlexGalhardo/vukem-trading.git
   cd vukem-trading
   git config core.hooksPath .githooks
   ```

2. Create `.env` from the template **only if it does not exist**, then generate the key. Never overwrite an existing key or configuration:

   ```sh
   cp -n .env.example .env
   vukem key:generate
   ```

3. Set the owner and Google credentials in `.env`:

   ```text
   OWNER_GOOGLE_EMAIL=you@example.com
   SOCIAL_GOOGLE_CLIENT_ID=...
   SOCIAL_GOOGLE_CLIENT_SECRET=...
   ```

4. Start everything:

   ```sh
   docker compose up --build
   ```

   - Web: <http://localhost:3000>
   - Mail catcher: <http://localhost:8025>
   - Engine health: `http://127.0.0.1:3001/ready` inside the network

   The web container applies the database migrations before the engine starts. Accounts begin blocked and strategies disabled; enable them explicitly after reviewing the limits.

## Google sign-in

Create an OAuth client of type Web application in Google Cloud and add the redirect URI `http://localhost:3000/auth/google/callback`. Google verifies the identity and the server admits only the account in `OWNER_GOOGLE_EMAIL`. Without OAuth configured, the login page says what is missing.

## Optional integrations

- `ALPACA_API_KEY` and `ALPACA_API_SECRET`: Alpaca **paper** credentials for the IEX history import. Live hosts cannot be configured.
- `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID`: operational alerts. Keep them out of any `VITE_*` variable.

## Run without containers for the app

```sh
bun install --frozen-lockfile
docker compose up -d postgres
vukem migrate
vukem serve
ENGINE_HEALTH_PORT=3001 go run ./cmd/engine
```

## Verify

```sh
bun run typecheck
bun run lint
go vet ./...
go test -race ./...
```

For PostgreSQL integration tests create a dedicated database whose name ends in `_test` and set `TEST_DATABASE_URL`. Never point tests at a shared or production database.

## Troubleshooting

- `docker compose up` fails on the web health check: read `docker compose logs app`; a missing `APP_KEY` is the usual cause.
- Login says OAuth is not configured: set the three Google variables and restart the web container.
- Port 3000 or 5432 is in use: stop the other process or change the published port in `compose.yaml`.

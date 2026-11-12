# syntax=docker/dockerfile:1
# Production image: Vite assets, a static Go binary, a small non-root runtime.
# Railway, Fly, Render and plain Docker build it as is:
# https://vukem.dev/docs/deployment/docker-and-railway

FROM oven/bun:1.4.4-alpine@sha256:4cf82f5686b196504fe119db3fcd9185b191f82ece82c40680e597228ec6070a AS assets
WORKDIR /src
COPY package.json bun.lock ./
# The lockfile is a required build input: --frozen-lockfile fails the build
# when bun.lock and package.json disagree, so the image installs exactly the
# versions the repository tracks. .dockerignore must not exclude bun.lock.
# Use ordinary layer caching: Railway cache mounts require a literal service ID,
# which would bind this shared Dockerfile to either the web or engine service.
RUN bun install --frozen-lockfile
COPY . .
RUN bun run build

FROM golang:1.27.4-alpine3.24@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS build
WORKDIR /src
ENV CGO_ENABLED=0
ARG RAILWAY_GIT_COMMIT_SHA=unknown
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=assets /src/resources/public/build resources/public/build
# Same tags as vukem build: .buildtags lists them (one per line or
# comma-separated, "#" comments); --minimal apps put vukem_nosqlite there.
RUN tags="$(sed 's/#.*//' .buildtags 2>/dev/null | tr -s ', \t\r\n' ',' | sed 's/^,//; s/,$//')" \
	&& go build -trimpath -ldflags="-s -w -X github.com/AlexGalhardo/vukem-trading/internal/trading.BuildVersion=$RAILWAY_GIT_COMMIT_SHA" -tags "$tags" -o /out/app ./cmd/app \
	&& go build -trimpath -ldflags="-s -w -X github.com/AlexGalhardo/vukem-trading/internal/trading.BuildVersion=$RAILWAY_GIT_COMMIT_SHA" -tags "$tags" -o /out/engine ./cmd/engine

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
RUN apk add --no-cache ca-certificates tzdata \
	&& adduser -D -H -u 10001 app \
	&& mkdir -p /app/storage \
	&& chown app:app /app/storage
WORKDIR /app
COPY --from=build /out/app /app/app
COPY --from=build /out/engine /app/engine
USER app
# The platform's PORT overrides this one; HOST=0.0.0.0 makes the server
# reachable from outside the container. Pass secrets at runtime, never here.
# MAIL_MAILER=log keeps the image runnable: with APP_ENV=production an unset
# mailer stops every command at boot, serve and `about` alike, as soon as a
# route needs one. "log" writes each message to the JSON log, so nothing is
# lost silently; set smtp, resend or mailtrap on the deploy to send for real.
ENV APP_ENV=production HOST=0.0.0.0 PORT=3000 LOG_FORMAT=json MAIL_MAILER=log
# A platform start command (Railway, Fly) replaces this entrypoint, so it has
# to name the binary: "/app/app queue:work", never "queue:work".
EXPOSE 3000
ENTRYPOINT ["/app/app"]

# OwnCall

OwnCall is a small, self-hosted audio/video calling service built with **Go, WebSocket signaling, and native WebRTC**. It is designed to be understandable, portable, and independent from LiveKit, Daily, Twilio, or other hosted media providers.

The current release uses peer-to-peer media for small rooms. The Go server coordinates the connection and serves the web client; it does not proxy media in the normal case.

> The software can be free and open source forever under MIT. A public service is not guaranteed to be free forever: a real deployment may need a machine, domain, TLS, bandwidth, and a TURN relay.

## Features

The repository includes a browser UI, room-based calls, microphone and camera controls, WebSocket signaling, runtime ICE configuration, Docker packaging, a Caddy HTTPS example, a coturn example, health checks, security headers, origin restrictions, message limits, graceful shutdown, tests, and GitHub Actions CI.

## Quick start

Requirements: Go 1.22 or newer and a modern browser.

```bash
export PATH=/usr/local/go/bin:$PATH
go mod download
go test ./...
go vet ./...
go run ./cmd/server
```

Open `http://localhost:8080` in two tabs, use the same room name, and grant camera/microphone permission. Localhost is treated as a secure context by browsers.

## Docker

```bash
cp .env.example .env
# Edit .env before a public deployment.
docker compose up --build
```

For production, put the service behind Caddy or another TLS reverse proxy. Browsers normally require HTTPS before allowing camera and microphone access on a non-localhost domain.

## TURN configuration

Direct WebRTC connections do not work for every network. For reliable internet connectivity, run your own coturn instance and set `ICE_SERVERS_JSON` in `.env`:

```env
ICE_SERVERS_JSON=[{"urls":["turn:turn.example.com:3478?transport=udp","turn:turn.example.com:3478?transport=tcp"],"username":"owncall","credential":"replace-with-a-strong-secret"}]
```

A coturn Docker example is in `deploy/coturn/docker-compose.yml`. Open UDP/TCP 3478 and the configured relay port range in the firewall. For a TURN server behind NAT, configure coturn's `--external-ip` mapping.

## HTTPS with Caddy

Copy `Caddyfile.example` to `Caddyfile`, replace the hostname, and run OwnCall and Caddy on the same Docker network. Caddy will manage the public certificate and proxy WebSocket upgrades automatically. Set `ALLOWED_ORIGINS` to the exact HTTPS origin, for example `https://call.example.com`.

## Architecture

The browser captures local media with `getUserMedia`, creates an `RTCPeerConnection` for each other participant, and exchanges SDP offers/answers and ICE candidates through `/ws`. The server keeps room membership in memory and routes only signaling messages. A restart intentionally clears all rooms.

This mesh design is suitable for one-to-one calls and small rooms. Each participant sends media to every other participant, so bandwidth and CPU usage grow with room size. Larger rooms require an SFU; that can be a future OwnCall component using Pion WebRTC, but it will consume server bandwidth and compute.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `ADDR` | `:8080` | HTTP bind address |
| `ALLOWED_ORIGINS` | same origin | Comma-separated WebSocket origins; use exact origins in production |
| `ICE_SERVERS_JSON` | `[]` | JSON array passed to the browser's WebRTC configuration |

## Security status

This repository is ready for a controlled self-hosted deployment, but it is not an identity provider. The default MVP does not authenticate users or persist room permissions. Before a public launch, add room access tokens or an application login, rate limiting at the proxy, monitoring, and a clear retention/privacy policy. See `SECURITY.md`.

## GitHub publishing

Create an empty GitHub repository, then run:

```bash
git remote add origin https://github.com/YOUR_GITHUB_USERNAME/owncall.git
git push -u origin main
```

Replace `github.com/yourname/owncall` in `go.mod` and imports with your actual repository path before publishing if you want consumers to import this module. The generated release archive should not contain `.git`, `.env`, credentials, or private TURN configuration.

## Project layout

- `cmd/server` — HTTP and WebSocket server.
- `internal/signaling` — in-memory room registry and message routing.
- `web` — browser client and styling.
- `deploy/coturn` — optional TURN deployment example.
- `Caddyfile.example` — HTTPS reverse-proxy example.
- `.github/workflows/ci.yml` — automated checks.

## License

MIT. See `LICENSE`.

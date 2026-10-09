# OwnCall

Self-hosted video/audio calls built with **Go + WebSocket signaling + native WebRTC**. The server does not carry media in the MVP: browsers establish peer-to-peer connections after exchanging SDP and ICE candidates.

> This project removes dependency on LiveKit-like media providers. It does **not** make global video calling free forever: a public deployment still needs a machine, domain/TLS, bandwidth, and usually a TURN relay for users behind restrictive NATs.

## Quick start

```bash
go mod tidy
go test ./...
go run ./cmd/server
```

Open http://localhost:8080 in two browser tabs, use the same room name, and allow camera/microphone access.

Docker:

```bash
docker compose up --build
```

## How it works

- `cmd/server`: HTTP server, static UI, WebSocket signaling endpoint at `/ws`.
- `internal/signaling`: in-memory room registry and message routing.
- `web`: browser client using `RTCPeerConnection`.
- Media is P2P in the current 1:1/small-room MVP; the Go server only relays signaling messages.

## Free/self-hosted reality

- **Code and license:** can be free forever (MIT).
- **Local/LAN use:** can be free using your own devices and network.
- **Public internet:** not guaranteed free forever. You need hosting and traffic; free tiers can change or impose limits.
- **NAT traversal:** direct P2P may fail. For reliable connectivity, run your own [coturn](https://github.com/coturn/coturn) TURN server and configure `iceServers` in `web/index.html` or inject it through a deployment-specific config.
- **TLS:** browsers require HTTPS for camera/microphone outside localhost. Use Caddy/Nginx with a certificate.
- **Scaling:** mesh calls are appropriate for small rooms. Group calls need an SFU; a future OwnCall SFU could be implemented with Pion WebRTC, but that requires server bandwidth and CPU.

## Roadmap

1. Authentication, room authorization, rate limits, and structured errors.
2. Self-hosted coturn configuration and HTTPS reverse-proxy examples.
3. Screen sharing, device selection, reconnection, and call quality metrics.
4. Optional SFU mode for larger rooms.
5. Persistent room metadata only if needed; keep media out of the database.

## Security before production

The current MVP intentionally has no authentication, room password, abuse protection, or persistent storage. Do not expose it publicly without adding those controls, HTTPS, origin restrictions, logging policy, and resource limits.

## License

MIT.

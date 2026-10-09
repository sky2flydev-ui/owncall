# Security policy

## Before public deployment

OwnCall is a signaling server and does not authenticate users in the default configuration. Before exposing it to the internet:

1. Put it behind HTTPS (for example Caddy).
2. Set `ALLOWED_ORIGINS` to the exact public origin; do not use `*`.
3. Run coturn with a strong, rotated credential and restrict firewall ports.
4. Add an application-level room access token or authentication layer for your use case.
5. Add rate limiting at the reverse proxy and monitor resource usage.
6. Do not log SDP, ICE credentials, room secrets, or personal data.
7. Keep Go, Docker, coturn, and the host OS patched.

## Reporting a vulnerability

Please do not open a public issue for an undisclosed security vulnerability. Contact the repository maintainers privately after replacing the placeholder contact in this file.

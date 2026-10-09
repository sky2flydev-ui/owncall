# Contributing

1. Create a branch from `main`.
2. Keep changes focused and documented.
3. Run the full checks before opening a pull request:

```bash
gofmt -w cmd internal
 go test ./...
 node --check web/app.js
 go vet ./...
```

4. Do not commit credentials, `.env` files, private TURN keys, or generated archives.
5. Describe browser, network, and deployment details for WebRTC bugs.

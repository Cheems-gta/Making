# Making

Making is an independent Go networking-core project, designed to run without the Doreena GUI or Android runtime.

## Standalone v0.1 — foundation preview

**This is a proof-of-concept TCP proxy runtime, not the full Android Making transport engine.**
The current published Go source supports:

- Authenticated loopback SOCKS5 CONNECT and HTTP CONNECT listeners
- TCP forwarding through an **existing upstream HTTP or SOCKS5 proxy**
- JSON configuration with explicitly required upstream (no silent DIRECT fallback)
- Windows x64 cross-compilation, Go unit and local end-to-end regression tests

**Not implemented in standalone v0.1:** Hysteria2, AnyTLS, TUIC, VLESS, VMess, UDP, TUN, YAML subscriptions, Windows system-proxy control or a GUI. A local READY message is not proof of working remote transport.

## Build from source

Go 1.23 or newer:

```bash
go test ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o Making-Standalone-Windows-x64.exe ./cmd/making
```

The GitHub Actions **Build Making Standalone Windows** workflow runs the tests and builds a Windows x64 EXE artifact on pushes to main or manual dispatch.

## Run

Copy `config.example.json` to `config.json`, set an appropriately strong local password and the URL of a trusted upstream HTTP/SOCKS5 proxy, then run:

```powershell
.\Making-Standalone-Windows-x64.exe -config config.json
```

Only loopback listeners are allowed. Keep `config.json` private; the repository's `.gitignore` excludes it. Never commit real subscriptions, passwords, tokens or signing keys.

## Cross-platform engine roadmap

- Integrate the established Go transport source and document upstream provenance and licensing
- Introduce common configuration and control API
- Port Hysteria2 and AnyTLS independently of Android's VPNService and gomobile
- Add Windows system-proxy integration and optional TUN
- Extend Linux/macOS/Android support after transport testing

The existing Android engine is in [Cheems-Core / making-latest](https://github.com/Cheems-gta/Cheems-Core/tree/making-latest); the Doreena frontend is in [Cheems-Clash](https://github.com/Cheems-gta/Cheems-Clash). These projects must not be confused with this standalone foundation.

## License notice

The historical Making transport tree includes GPL-covered code and third-party dependencies. This standalone foundation is published as source; a definitive redistribution license for project-authored files must be confirmed with the rights holder(s). Do not assume importing GPL-covered upstream code changes its license. Preserve applicable upstream notices when porting code.

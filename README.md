# Making

Making is an independent, open-source networking core project written in Go.

## Project status

**Early preview — not a production-ready VPN or a verified Hysteria2/AnyTLS Windows release.**

The Windows Standalone v0.1 preview is a separate preliminary TCP forwarding implementation. It supports local HTTP CONNECT / SOCKS5 interfaces and forwarding through an existing upstream HTTP or SOCKS5 proxy. It does **not** yet include the latest Android Making Hysteria2 / AnyTLS transports.

## Roadmap

- [ ] Consolidate the Making Go transport source with clear provenance and upstream notices
- [ ] Make the core usable without Android, a GUI or Doreena
- [ ] Build and test Windows x64 binaries
- [ ] Integrate Hysteria2 and AnyTLS using the maintained transport code
- [ ] Implement YAML configuration and a local control API
- [ ] Add Linux and Android adapters
- [ ] Perform protocol interoperability, security and reliability testing

## Existing source

The prior Android native engine and source patches are maintained in the [Cheems-Core making-latest branch](https://github.com/Cheems-gta/Cheems-Core/tree/making-latest). The Android Doreena frontend is maintained separately in [Cheems-Clash](https://github.com/Cheems-gta/Cheems-Clash).

## Security and license

Never commit subscriptions, credentials, certificates containing private keys, or real proxy configuration. Existing Making branches include historical GPL licensing and third-party code with their own obligations. Review copyright and redistribution rights before assigning a new license to imported source. This repository's README does not change the terms of upstream source.

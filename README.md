# ssh-pubkey-server

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/ssh-pubkey-server)](https://goreportcard.com/report/github.com/flashbots/go-template)
[![Test status](https://github.com/flashbots/ssh-pubkey-server/workflows/Checks/badge.svg?branch=main)](https://github.com/flashbots/go-template/actions?query=workflow%3A%22Checks%22)

---

## Getting started

**Run CLI**

The following will request server ssh pubkey through a proxy, and separately run ssh-keyscan and will return the matching server keys that you can then append to your known_hosts.  

```bash
./cmd/cli/add_to_known_hosts.sh <attested http proxy> <host ip> >> ~/.ssh/known_hosts
```

**Build HTTP server**

```bash
make build-httpserver
```

**Run pubkey server**

```bash
go run ./cmd/httpserver/main.go [--listen-addr=127.0.0.1:8080] [--host-ssh-pubkey-file=/etc/ssh/ssh_host_ed25519_key.pub] [--container-ssh-pubkey-file=/path/to/container_key.pub]
```

The server will serve both pubkeys (if provided) at the `/pubkey` endpoint, separated by a newline. The `--container-ssh-pubkey-file` flag is optional.

**Install dev dependencies**

```bash
go install mvdan.cc/gofumpt@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/daixiang0/gci@latest
```

**Lint, test, format**

```bash
make lint
make test
make fmt
```

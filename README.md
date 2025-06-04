# ssh-pubkey-server

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/ssh-pubkey-server)](https://goreportcard.com/report/github.com/flashbots/go-template)
[![Test status](https://github.com/flashbots/ssh-pubkey-server/workflows/Checks/badge.svg?branch=main)](https://github.com/flashbots/go-template/actions?query=workflow%3A%22Checks%22)

---

## Getting started

**Run CLI**

The following will request server ssh pubkey through a proxy, and separately run ssh-keyscan and will return the matching server keys that you can then append to your known_hosts.

```bash
$ bash ./cmd/cli/add_to_known_hosts.sh --help
Makes sure the pubkey returned from proxy matches ssh-keyscan of the host, and formats in a way that can be appended to known_hosts.
Usage:	./add_to_known_hosts.sh [--proxy=http://127.0.0.1:8080] --ssh-host=<hosts ip> [--ssh-port=22] >> ~/.ssh/known_hosts
	Make sure your cvm-reverse-proxy client is running.
```

**Build HTTP server**

```bash
make build-httpserver
```

**Run pubkey server**

```bash
go run ./cmd/httpserver/main.go [--listen-addr=127.0.0.1:8080] [--ssh-pubkey-file=/etc/ssh/ssh_host_ed25519_key.pub]
```

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

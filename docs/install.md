# Installation

## Requirements

- Go 1.25 or later

## Install the binary

```bash
go install github.com/jonbaldie/go-mutesting/v2/cmd/go-mutesting@latest
```

The binary is placed in `$(go env GOPATH)/bin`. Make sure that directory is on your `PATH`.

## Build from source

```bash
git clone https://github.com/jonbaldie/go-mutesting.git
cd go-mutesting
go build -o go-mutesting ./cmd/go-mutesting
```

## Verify

```bash
go-mutesting --help
```

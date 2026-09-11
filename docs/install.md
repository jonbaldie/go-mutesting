# Installation

## Requirements

- Go 1.26 or later

## Install the binary

```bash
go install github.com/jonbaldie/go-mutesting/v2/cmd/go-mutesting@latest
go-mutesting --version
```

The binary is placed in `$(go env GOPATH)/bin`. Make sure that directory is on your `PATH`.

## Docker

Production image (runs go-mutesting against a mounted module):

```bash
docker build -t go-mutesting .
docker run --rm -v "$PWD":/code -w /code go-mutesting ./...
```

Development image (Go toolchain plus project sources for local iteration):

```bash
docker build -f dev.Dockerfile -t go-mutesting-dev .
docker run --rm -it -v "$PWD":/workspace go-mutesting-dev
```

## Build from source

```bash
git clone https://github.com/jonbaldie/go-mutesting.git
cd go-mutesting
go build -o go-mutesting ./cmd/go-mutesting
```

## Verify

```bash
go-mutesting --version
```

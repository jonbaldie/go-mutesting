# Development image: docker build -f dev.Dockerfile -t go-mutesting-dev . && docker run --rm -it -v "$PWD":/workspace go-mutesting-dev
FROM golang:alpine
RUN apk add --no-cache git diffutils make gcc musl-dev bash
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["go", "test", "./..."]

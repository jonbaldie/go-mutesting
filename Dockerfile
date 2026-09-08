# Runtime image: docker build -t go-mutesting . && docker run --rm -v "$PWD":/code -w /code go-mutesting ./...
FROM golang:alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app/go-mutesting ./cmd/go-mutesting

FROM golang:alpine
RUN apk add --no-cache git diffutils ca-certificates
COPY --from=build /app/go-mutesting /usr/local/bin/go-mutesting
WORKDIR /code
ENTRYPOINT ["go-mutesting"]
CMD ["--help"]

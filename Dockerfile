FROM golang:1.20 AS builder

WORKDIR /opt/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -a -installsuffix cgo \
    -o bin/amd64/scratch/baraddur

# --------
FROM scratch

WORKDIR /

COPY --from=builder /opt/app/bin/amd64/scratch .

ENTRYPOINT [ "/app" ]
CMD ["--help"]
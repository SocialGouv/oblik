FROM golang:1.22 AS builder

WORKDIR /app

ARG VERSION=dev

COPY go.mod go.sum ./

COPY vendor vendor

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -mod=vendor -ldflags="-X 'github.com/SocialGouv/oblik/pkg/cli.Version=${VERSION}'" -o oblik .

FROM alpine:3 AS certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/oblik /oblik

USER 1000
ENTRYPOINT ["/oblik"]
CMD ["operator"]

FROM golang:1.22 as builder

WORKDIR /app

COPY go.mod go.sum ./

COPY vendor vendor

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -mod=vendor -o oblik .

FROM alpine:3 as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssh/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/oblik /oblik

USER 1000
ENTRYPOINT ["/oblik"]

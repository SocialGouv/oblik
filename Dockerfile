FROM golang:1.22 as builder

WORKDIR /app

COPY go.mod go.sum ./

COPY vendor vendor

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -mod=vendor -o oblik .


FROM scratch

COPY --from=builder /app/oblik /oblik

ENTRYPOINT ["/oblik"]

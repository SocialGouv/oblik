all: build

.PHONY: build
build: preflight
	CGO_ENABLED=0 go build -a -installsuffix cgo -mod=vendor -o oblik .

.PHONY: preflight
preflight:
	go mod vendor
	go fmt github.com/SocialGouv/oblik

update:
	go mod tidy
	go mod vendor

docker-build:
	docker build . -t ghcr.io/socialgouv/oblik

docker-push:
	docker push ghcr.io/socialgouv/oblik

docker: docker-build docker-push

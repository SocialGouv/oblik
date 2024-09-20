all: build

.PHONY: build
build: preflight
	go build -mod=vendor

.PHONY: build-static
build-static: preflight
	CGO_ENABLED=0 go build -a -installsuffix cgo -mod=vendor -o oblik .


.PHONY: preflight
preflight: deps
	go fmt github.com/SocialGouv/oblik

deps:
	go mod tidy
	go mod vendor

update-deps:
	go get -u ./...

docker-build:
	docker build . -t ghcr.io/socialgouv/oblik

docker-push:
	docker push ghcr.io/socialgouv/oblik

docker: docker-build docker-push

setup-test-env:
	./tests/kind-with-registry.sh
	./tests/install-dependencies.sh

test:
	./tests/deploy-oblik.sh
	./tests/test-oblik.sh $(ARGS)
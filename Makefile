.PHONY: test
ROOT_DIR = $(shell pwd)
GOPATH:=$(shell go env GOPATH)
USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)

#################################################################################
# RUN COMMANDS
#################################################################################
run:
	go mod vendor
	docker-compose --file ./build/docker-compose.yml --file ./build/docker-compose.dev.yml --project-directory . pull
	docker-compose --file ./build/docker-compose.yml --file ./build/docker-compose.dev.yml --project-directory . up --build; \
	docker-compose --file ./build/docker-compose.yml --file ./build/docker-compose.dev.yml --project-directory . down --volumes; \
	rm -rf vendor

delve:
	go mod vendor
	docker-compose --file ./build/docker-compose.yml --file ./build/docker-compose.delve.yml --project-directory . up --build --abort-on-container-exit --exit-code-from gpsi-migration-vehicles-svc; \
	docker-compose --file ./build/docker-compose.yml --file ./build/docker-compose.delve.yml --project-directory . down --volumes; \
	rm -rf vendor

#################################################################################
# LINT COMMANDS
#################################################################################
tidy:
	goimports -w .
	gofmt -s -w .
	go mod tidy

#################################################################################
# TEST COMMANDS
#################################################################################
test:
	go test -cover ./...
	golangci-lint run

test-coverage:
	go test -coverpkg ./... -coverprofile coverage.out ./... && go tool cover -html=coverage.out

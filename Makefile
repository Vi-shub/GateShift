APP := gateshift
OPERATOR := gateshift-operator
PKG := ./...
BIN_DIR := bin
CORPUS := examples/corpus
SCOREBOARD_OUT := docs/scoreboard.md

.PHONY: all build build-operator test lint fmt tidy clean scoreboard run-audit run-convert run-dual-run run-validate run-migrate helm-lint

all: tidy test build build-operator

build:
	go build -o $(BIN_DIR)/$(APP)$(shell go env GOEXE) ./cmd/gateshift

build-operator:
	go build -o $(BIN_DIR)/$(OPERATOR)$(shell go env GOEXE) ./cmd/gateshift-operator

test:
	go test $(PKG) -count=1

tidy:
	go mod tidy

fmt:
	gofmt -w .

helm-lint:
	helm lint charts/gateshift-operator
	helm template gateshift-operator charts/gateshift-operator >/dev/null

scoreboard: build
	$(BIN_DIR)/$(APP)$(shell go env GOEXE) scoreboard -f $(CORPUS) -o $(SCOREBOARD_OUT)

clean:
	rm -rf $(BIN_DIR) .gateshift-pr .gateshift-e2e

run-audit: build
	$(BIN_DIR)/$(APP)$(shell go env GOEXE) audit -f examples/ingress-checkout.yaml --target=envoy-gateway

run-convert: build
	$(BIN_DIR)/$(APP)$(shell go env GOEXE) convert -f examples/ingress-checkout.yaml --target=envoy-gateway

run-dual-run: build
	$(BIN_DIR)/$(APP)$(shell go env GOEXE) dual-run -f examples/ingress-checkout.yaml --target=envoy-gateway

run-validate: build
	$(BIN_DIR)/$(APP)$(shell go env GOEXE) validate -f examples/ingress-checkout.yaml --target=envoy-gateway

run-migrate: build
	$(BIN_DIR)/$(APP)$(shell go env GOEXE) migrate -f examples/ingress-checkout.yaml --target=envoy-gateway

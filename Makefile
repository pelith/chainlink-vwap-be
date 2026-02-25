.PHONY: build docker docker-up release lint gci-format test coverage gen-migration-sql sqlc abigen
NAME=vwap

build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o "${NAME}_$(app)" ./cmd/$(app)

app?=api
tag?=latest

docker:
	docker build --platform linux/amd64 -t $(NAME):$(tag) .

docker-fast:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o main ./cmd/api
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o migration ./cmd/migration
	docker build -f Dockerfile.fast --platform linux/amd64 -t $(NAME):$(tag) .
	@rm -f main migration

docker-up:
	docker compose up

docker-up-fast: docker-fast
	docker compose up

TAG := v$(shell date -u '+%Y.%m.%d.%H.%M.%S')

release: 
	git tag $(TAG)
	git push origin $(TAG)

# Tools

lint:
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint run ./... -c ./.golangci.yml

lint-fix:
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint run ./... -c ./.golangci.yml --fix

gci-format:
	@go run github.com/daixiang0/gci write --skip-generated -s standard -s default -s "Prefix(vwap)" ./

test:
	@go test ./... -race  

coverage:
	@go test -coverprofile=coverage.out ./internal/...
	@go tool cover -func=coverage.out

# SQL

DATETIME=$(shell date -u '+%Y%m%d%H%M%S')

gen-migration-sql:
	@( \
	printf "Enter file name: "; read -r FILE_NAME; \
	touch database/migrations/$(DATETIME)_$$FILE_NAME.up.sql; \
	touch database/migrations/$(DATETIME)_$$FILE_NAME.down.sql; \
	)

gen-seed-sql:
	@( \
	printf "Enter file name: "; read -r FILE_NAME; \
	printf "Enter env: "; read -r ENV; \
	mkdir -p database/seeds/$$ENV; \
	touch database/seeds/$$ENV/$(DATETIME)_$$FILE_NAME.up.sql; \
	touch database/seeds/$$ENV/$(DATETIME)_$$FILE_NAME.down.sql; \
	)

sqlc:
	sqlc generate -f ./database/sqlc.yml

sqlc-lint:
	sqlc vet -f ./database/sqlc.yml

# Contracts (require go-ethereum abigen: go install github.com/ethereum/go-ethereum/cmd/abigen@latest)

# VWAP RFQ Spot: generate Go binding from contract/abi/VWAPRFQSpot.json
abigen-vwap:
	@mkdir -p internal/contracts/vwaprfqspot && abigen --abi contract/abi/VWAPRFQSpot.json --pkg vwaprfqspot --type VWAPRFQSpot --out internal/contracts/vwaprfqspot/binding.go

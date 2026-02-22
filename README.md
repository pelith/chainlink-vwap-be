# vwap

## Run locally

### 1. Start the database

```bash
docker compose up -d postgres
```

Uses `config/api` settings: host `127.0.0.1`, port `5432`, user `postgres`, password `postgres`, database `vwap_local`.

### 2. Run migrations

```bash
ENV=local go run ./cmd/migration
```

Reads `config/migration` and runs migrations + seeds against `vwap_local`. If the database does not exist, it is created first (connects to `postgres` then `CREATE DATABASE vwap_local`).

### 3. Run the API

```bash
ENV=local go run ./cmd/api
```

Or build then run:

```bash
make build app=api
./vwap_api
```

API listens on **http://127.0.0.1:8080**. Example:

```bash
curl http://127.0.0.1:8080/users/{user-uuid}
```

Requires existing user data (from seed or manual insert).

---

## Requirment

### go
> always use the latest version of go, now is 1.24.4
```bash
brew intall go
```

### cocogitto
for [Conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) 
[command usage](https://docs.cocogitto.io/guide/commit.html)
```bash
brew install cocogitto
```

### other tools
```bash
go install tool
```

after `go install tool`, install lefthook git hooks
```bash
lefthook install
```
# go-template

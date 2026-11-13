VAULT_HOST ?= 192.168.1.194
SSH_OPTS   ?= -o StrictHostKeyChecking=accept-new -o BatchMode=yes

.PHONY: help fetch-ca vet build test test-short test-stepdown

help:
	@echo "make fetch-ca        75-vault-cluster のサーバーから CA 証明書を certs/ca.crt に取得"
	@echo "make test            全テスト (VAULT_TOKEN が必要。クライアント VM では vault login 済みなら不要)"
	@echo "make test-short      時間のかかるテスト (トークン更新、約 15 秒) を省略"
	@echo "make test-stepdown   リーダー交代のテストも実行 (アクティブノードがステップダウンする)"
	@echo "make build           kvdemo (サンプルプログラム) を bin/ にビルド"

fetch-ca:
	mkdir -p certs
	scp $(SSH_OPTS) ubuntu@$(VAULT_HOST):/opt/vault/tls/ca.crt certs/ca.crt

vet:
	go vet ./...

build:
	go build -o bin/kvdemo ./cmd/kvdemo

test:
	go test -count=1 -v -timeout 10m ./...

test-short:
	go test -count=1 -short -v -timeout 10m ./...

test-stepdown:
	VAULT_TEST_STEPDOWN=1 go test -count=1 -v -timeout 10m -run 'TestLeaderStepDown|TestClusterHealth' ./...

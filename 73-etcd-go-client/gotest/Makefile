# etcd-use で設定済みの ETCD_TARGET があればそれを使う。別の接続先を指定した場合は ETCD_ENDPOINTS を引き継がない
TARGET ?= $(if $(ETCD_TARGET),$(ETCD_TARGET),single)
TEST_ENV = ETCD_TARGET=$(TARGET) ETCD_ENDPOINTS="$(if $(filter $(TARGET),$(ETCD_TARGET)),$(ETCD_ENDPOINTS),)"
SINGLE_HOST  ?= 192.168.1.190
CLUSTER_HOST ?= 192.168.1.191
SSH_OPTS     ?= -o StrictHostKeyChecking=accept-new -o BatchMode=yes

.PHONY: help fetch-ca test test-short test-all test-compact vet

help:
	@echo "make fetch-ca                 71/72 のサーバーから CA 証明書を certs/ に取得"
	@echo "make test TARGET=single       71-etcd-single に対してテスト"
	@echo "make test TARGET=cluster      72-etcd-cluster に対してテスト"
	@echo "make test-all                 single と cluster の両方でテスト"
	@echo "make test-short TARGET=...    時間のかかるテストを省略"
	@echo "make test-compact TARGET=...  compaction のテストも実行 (etcd の履歴を圧縮する)"

fetch-ca:
	mkdir -p certs
	scp $(SSH_OPTS) ubuntu@$(SINGLE_HOST):/etc/etcd/pki/ca.crt certs/single-ca.crt
	scp $(SSH_OPTS) ubuntu@$(CLUSTER_HOST):/etc/etcd/pki/ca.crt certs/cluster-ca.crt

vet:
	go vet ./...

test:
	$(TEST_ENV) go test -count=1 -v -timeout 10m ./...

test-short:
	$(TEST_ENV) go test -count=1 -short -v -timeout 10m ./...

test-all:
	$(MAKE) test TARGET=single
	$(MAKE) test TARGET=cluster

test-compact:
	ETCD_TEST_COMPACT=1 $(TEST_ENV) go test -count=1 -v -timeout 10m -run TestWatchRecoversFromCompaction ./...

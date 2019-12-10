VERSION=$(shell git describe --tags)

# DEPLOYTO must be set to hostname to deploy by ssh.
DEPLOYBINARY=ruuvi-prometheus.armv6-unknown-linux
DEPLOYPATH=/usr/local/bin/ruuvi-prometheus
PREFIX = /usr/local

.PHONY: ruuvi-prometheus
ruuvi-prometheus:
	go build -o $@ -ldflags="-s -w -X main.version=${VERSION}"

build: ruuvi-prometheus

.PHONY: clean
clean:
	rm -f ruuvi-prometheus

.PHONY: test
test:
	go test -race ./...

.PHONY: install
install: ruuvi-prometheus
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp $< $(DESTDIR)$(PREFIX)/bin/ruuvi-prometheus

.PHONY: deploy
## deploy: quick deploy to raspberry pi, replacing existing installed version
deploy: export GOOS = linux
deploy: export GOARCH = arm
deploy: export GOARM = 6
deploy: export CGO_ENABLED = 0
deploy: check-deployto ruuvi-prometheus
	ssh $(DEPLOYTO) /etc/init.d/ruuvi-prometheus stop
	scp ruuvi-prometheus $(DEPLOYTO):$(DEPLOYPATH)
	ssh $(DEPLOYTO) /etc/init.d/ruuvi-prometheus start
	@sleep 3
	curl -s http://$(DEPLOYTO):9521/metrics|grep temperature

.PHONY: check-deployto
check-deployto:
ifndef DEPLOYTO
	$(error No deploy target set. Please set DEPLOYTO to hostname to deploy to.)
endif


ALPINE_TARBALL := http://dl-cdn.alpinelinux.org/alpine/v3.8/releases/aarch64/alpine-rpi-3.8.2-aarch64.tar.gz

alpine-rpi-3.8.2-aarch64.tar.gz:
	curl -sL -o "$@" "$(ALPINE_TARBALL)"
	shasum -c SHA256SUM

root/usr/local/bin/ruuvi-prometheus:
	GOOS=linux GOARCH=arm go build -ldflags="-s -w" github.com/joneskoo/ruuvi-prometheus

.PHONY: build
build: alpine-rpi-3.8.2-aarch64.tar.gz root/usr/local/bin/ruuvi-prometheus
	rm -rf build
	mkdir build
	tar -C build -zxf alpine-rpi-3.8.2-aarch64.tar.gz


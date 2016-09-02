-include artifacts/build/Makefile.in

.PHONY: run
run: build artifacts/certificates
	CERTIFICATE_PATH=artifacts/certificates \
		$(BUILD_PATH)/debug/$(CURRENT_OS)/$(CURRENT_ARCH)/honeycomb

.PHONY: docker
docker: artifacts/docker.touch

artifacts/certificates:
	@mkdir -p "$@"
	openssl genrsa -out "$@/ca.key" 2048
	openssl genrsa -out "$@/server.key" 2048
	openssl req \
		-new \
		-x509 \
		-sha256 \
		-days 3650 \
		-extensions v3_ca \
		-key "$@/ca.key" \
		-out "$@/ca.crt" \
		-subj "/CN=Honeycomb Development CA"

artifacts/docker.touch: Dockerfile $(BUILD_PATH)/release/linux/amd64/honeycomb artifacts/certificates
	docker build -t honeycomb:dev .
	touch "$@"

artifacts/build/Makefile.in:
	mkdir -p "$(@D)"
	curl -Lo "$(@D)/runtime.go" https://raw.githubusercontent.com/icecave/make/master/go/runtime.go
	curl -Lo "$@" https://raw.githubusercontent.com/icecave/make/master/go/Makefile.in

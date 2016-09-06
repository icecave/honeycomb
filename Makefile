DOCKER_REPO ?= icecave/honeycomb
DOCKER_TAG  ?= dev

PREREQUISITES := $(patsubst res/assets/%,artifacts/assets/%.go, $(wildcard res/assets/*))

-include artifacts/build/Makefile.in

.PHONY: run
run: build artifacts/certificates
	CERTIFICATE_PATH=artifacts/certificates \
		$(BUILD_PATH)/debug/$(CURRENT_OS)/$(CURRENT_ARCH)/honeycomb \
		$$HONEYCOMB_ARGS

.PHONY: docker
docker: artifacts/docker.touch

.PHONY: deploy
deploy: docker
	docker push "$(DOCKER_REPO):$(DOCKER_TAG)"

artifacts/assets/%.go: res/assets/%
	@mkdir -p "$(@D)"
	@echo "package assets" > "$@"
	@echo >> "$@"
	@echo 'const Asset_$(subst .,_,$(notdir $<)) = `' >> "$@"
	cat "$<" >> "$@"
	@echo '`' >> "$@"

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
	docker build -t "$(DOCKER_REPO):$(DOCKER_TAG)" .
	touch "$@"

artifacts/build/Makefile.in:
	mkdir -p "$(@D)"
	curl -Lo "$(@D)/runtime.go" https://raw.githubusercontent.com/icecave/make/master/go/runtime.go
	curl -Lo "$@" https://raw.githubusercontent.com/icecave/make/master/go/Makefile.in

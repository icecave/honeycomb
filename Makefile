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

.PHONY: docker-services
docker-services: docker
	-docker service rm honeycomb honeycomb-echo
	docker service create \
		--name honeycomb \
		--publish 443:8443 \
		--constraint node.role==manager \
		--mount type=bind,target=/var/run/docker.sock,source=/var/run/docker.sock \
		icecave/honeycomb:dev
	docker service create \
		--name honeycomb-echo \
		--network ingress \
		--label 'honeycomb.match=echo.*' \
		jmalloc/echo-server

.PHONY: docker-logs
docker-logs:
	docker logs -f $(shell docker ps -qf label=com.docker.swarm.service.name=honeycomb)

MINIFY := $(GOPATH)/bin/minify
$(MINIFY):
	go get -u github.com/tdewolff/minify/cmd/minify

artifacts/assets/%.tmp: res/assets/% | $(MINIFY)
	$(MINIFY) -o "$@" "$<"

artifacts/assets/%.go: artifacts/assets/%.tmp
	@mkdir -p "$(@D)"
	@echo "package assets" > "$@"
	@echo 'const $(shell echo $(basename $*) | tr [:lower:] [:upper:]) = `' >> "$@"
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

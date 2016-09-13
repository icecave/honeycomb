DOCKER_REPO ?= icecave/honeycomb
DOCKER_TAG  ?= dev

CGO_ENABLED = 0

PREREQUISITES := $(patsubst res/assets/%,artifacts/assets/%.go, $(wildcard res/assets/*))
CERTIFICATES := $(addprefix artifacts/certificates/,ca.crt ca.key server.crt server.key)
-include artifacts/build/Makefile.in

.PHONY: run
run: $(BUILD_PATH)/debug/$(CURRENT_OS)/$(CURRENT_ARCH)/honeycomb $(CERTIFICATES)
	CERTIFICATE_PATH=$(CERTIFICATE_PATH:artifacts/certificates) \
		$(BUILD_PATH)/debug/$(CURRENT_OS)/$(CURRENT_ARCH)/honeycomb

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
		--env CERTIFICATE_S3_BUCKET="$$CERTIFICATE_S3_BUCKET" \
		--env CERTIFICATE_PATH="/" \
		--env AWS_ACCESS_KEY_ID="$$AWS_ACCESS_KEY_ID" \
		--env AWS_SECRET_ACCESS_KEY="$$AWS_SECRET_ACCESS_KEY" \
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
	$(MINIFY) -o "$@" "$<" || cp "$<" "$@"

artifacts/assets/%.go: artifacts/assets/%.tmp
	@mkdir -p "$(@D)"
	@echo "package assets" > "$@"
	@echo 'const $(shell echo $(notdir $*) | tr [:lower:] [:upper:] | tr . _) = `' >> "$@"
	cat "$<" >> "$@"
	@echo '`' >> "$@"

artifacts/certificates/%.key:
	@mkdir -p "$(@D)"
	openssl genrsa -out "$@" 2048

artifacts/certificates/%.csr.tmp: artifacts/certificates/%.key
	openssl req \
		-new \
		-sha256 \
		-subj "/CN=Honeycomb Default Certificate/subjectAltName=DNS.1=*" \
		-key "$<" \
		-out "$@"

artifacts/certificates/ca.crt: artifacts/certificates/ca.key
	openssl req \
		-new \
		-x509 \
		-sha256 \
		-days 30 \
		-extensions v3_ca \
		-nodes \
		-subj "/CN=Honeycomb CA" \
		-key "$<" \
		-out "$@"

artifacts/certificates/%.crt: artifacts/certificates/%.csr.tmp artifacts/certificates/extensions.cnf.tmp artifacts/certificates/ca.crt artifacts/certificates/ca.key
	openssl x509 \
		-req \
		-sha256 \
		-days 30 \
		-extfile artifacts/certificates/extensions.cnf.tmp \
		-CA artifacts/certificates/ca.crt \
		-CAkey artifacts/certificates/ca.key \
		-CAcreateserial \
		-in "$<" \
		-out "$@"

artifacts/certificates/extensions.cnf.tmp:
	echo "extendedKeyUsage = serverAuth" > "$@"

artifacts/docker.touch: Dockerfile $(addprefix $(BUILD_PATH)/release/linux/amd64/,$(BINARIES))
	docker build -t "$(DOCKER_REPO):$(DOCKER_TAG)" .
	touch "$@"

artifacts/build/Makefile.in:
	mkdir -p "$(@D)"
	curl -Lo "$(@D)/runtime.go" https://raw.githubusercontent.com/icecave/make/master/go/runtime.go
	curl -Lo "$@" https://raw.githubusercontent.com/icecave/make/master/go/Makefile.in

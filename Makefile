DOCKER_REPO ?= icecave/honeycomb
DOCKER_TAG  ?= dev

GENERATED_FILES += $(patsubst res/assets/%,artifacts/assets/%.go, $(wildcard res/assets/*))
DOCKER_BUILD_REQ += artifacts/cacert.pem

-include .makefiles/Makefile
-include .makefiles/pkg/go/v1/Makefile
-include .makefiles/pkg/docker/v1/Makefile

.PHONY: docker-services
docker-services: docker
	-docker service rm honeycomb honeycomb-echo
	-docker network create --driver=overlay public
	docker service create \
		--name honeycomb \
		--publish 80:8080 \
		--publish 443:8443 \
		--constraint node.role==manager \
		--mount type=bind,target=/var/run/docker.sock,source=/var/run/docker.sock \
		--secret honeycomb-ca.crt \
		--secret honeycomb-ca.key \
		--secret honeycomb-server.crt \
		--secret honeycomb-server.key \
		--network public \
		icecave/honeycomb:dev
	docker service create \
		--name honeycomb-echo \
		--network public \
		--label 'honeycomb.match=echo.*' \
		jmalloc/echo-server

.PHONY: docker-update
docker-update: docker
	docker service update --image icecave/honeycomb:dev --force honeycomb

.PHONY: docker-minio
docker-minio: docker-services
	-@mkdir -p artifacts/minio/data
	-docker service rm minio
	docker service create \
		--name minio \
		--constraint "node.role==manager" \
		--mount "type=bind,target=/data,source=$(shell pwd)/artifacts/minio/data" \
		--network public \
		--env "MINIO_ACCESS_KEY=TESTACCESSKEY" \
		--env "MINIO_SECRET_KEY=TESTSECRETKEY" \
		--publish "9000:9000/tcp" \
		minio/minio:latest \
		server /data
	docker service update \
		--env-add "MINIO_ACCESS_KEY=TESTACCESSKEY" \
		--env-add "MINIO_SECRET_KEY=TESTSECRETKEY" \
		honeycomb

MINIFY := artifacts/minify/bin/minify
$(MINIFY):
	-@mkdir -p "$(@D)"
	curl -sSL -# "https://github.com/tdewolff/minify/releases/download/v2.8.0/minify_$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH).tar.gz" | tar xvfz - -C "$(@D)" "minify"

artifacts/assets/%.txt.tmp: res/assets/%.txt
	-@mkdir -p "$(@D)"
	cp "$(<)" "$(@)"

artifacts/assets/%.tmp: res/assets/% | $(MINIFY)
	-@mkdir -p "$(@D)"
	$(MINIFY) -o "$@" "$<" || cp "$<" "$@"

.DELETE_ON_ERROR: artifacts/assets/%.go
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

artifacts/certificates/honeycomb-ca.crt: artifacts/certificates/honeycomb-ca.key
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

artifacts/certificates/%.crt: artifacts/certificates/%.csr.tmp artifacts/certificates/extensions.cnf.tmp artifacts/certificates/honeycomb-ca.crt artifacts/certificates/honeycomb-ca.key
	openssl x509 \
		-req \
		-sha256 \
		-days 30 \
		-extfile artifacts/certificates/extensions.cnf.tmp \
		-CA artifacts/certificates/honeycomb-ca.crt \
		-CAkey artifacts/certificates/honeycomb-ca.key \
		-CAcreateserial \
		-in "$<" \
		-out "$@"

artifacts/certificates/extensions.cnf.tmp:
	echo "extendedKeyUsage = serverAuth" > "$@"

artifacts/cabundle/gd_bundle-g2-g1.crt:
	@mkdir -p "$(@D)"
	curl -L -o "$@" https://certs.godaddy.com/repository/gd_bundle-g2-g1.crt

artifacts/cabundle/COMODO_DV_SHA-256_bundle.crt:
	-@mkdir -p "$(@D)"
	curl -SL -o "$(@)" "https://support.comodo.com/index.php?/Knowledgebase/Article/GetAttachment/1099/1226060"

artifacts/cacert.pem: artifacts/cabundle/gd_bundle-g2-g1.crt artifacts/cabundle/COMODO_DV_SHA-256_bundle.crt
	curl -L -o "$@.orig" http://curl.haxx.se/ca/cacert.pem
	cat "$@.orig" > "$@"
	( echo ""; echo "Go Daddy Intermediates"; echo "=================="; cat artifacts/cabundle/gd_bundle-g2-g1.crt ) >> "$@"
	( echo ""; echo "Comodo PositiveSSL Intermediates"; echo "=================="; cat artifacts/cabundle/COMODO_DV_SHA-256_bundle.crt ) >> "$@"

.makefiles/%:
	@curl -sfL https://makefiles.dev/v1 | bash /dev/stdin "$@"

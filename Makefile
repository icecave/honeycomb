DOCKER_REPO ?= icecave/honeycomb

MATRIX_OS ?= darwin linux
MATRIX_ARCH ?= amd64

CERTIFICATES := $(addprefix artifacts/certificates/honeycomb-,ca.crt ca.key server.crt server.key) artifacts/cacert.pem
CERTIFICATE_PATH ?= artifacts/certificates

REQ := $(patsubst res/assets/%,artifacts/assets/%.go, $(wildcard res/assets/*))
DOCKER_REQ := $(CERTIFICATES)

-include artifacts/make/go/Makefile
-include artifacts/make/docker/Makefile

artifacts/make/%/Makefile:
	curl -sf https://jmalloc.github.io/makefiles/fetch | bash /dev/stdin $*

.PHONY: run
run: artifacts/build/debug/$(GOOS)/$(GOARCH)/honeycomb $(CERTIFICATES)
	CERTIFICATE_PATH=$(CERTIFICATE_PATH) $(<) $(RUN_ARGS)

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

artifacts/certificates/honeycomb-ca.crt: artifacts/certificates/honeycomb-ca.key artifacts/certificates/openssl.cnf
	openssl req \
		-new \
		-x509 \
		-sha256 \
		-days 30 \
		-config artifacts/certificates/openssl.cnf \
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
	@mkdir -p "$(@D)"
	echo "extendedKeyUsage = serverAuth" | tee "$(@)"

artifacts/certificates/openssl.cnf:
	@mkdir -p "$(@D)"
	curl -sSL "https://raw.githubusercontent.com/openssl/openssl/master/apps/openssl.cnf" | tee "$(@)"

artifacts/cabundle/gd_bundle-g2-g1.crt:
	@mkdir -p "$(@D)"
	curl -L -o "$@" https://certs.godaddy.com/repository/gd_bundle-g2-g1.crt

artifacts/cabundle/comodo_dv_sha-256_bundle.crt.zip:
	@mkdir -p "$(@D)"
	curl -L -o "$@" https://namecheap.simplekb.com/SiteContents/2-7C22D5236A4543EB827F3BD8936E153E/media/COMODO_DV_SHA-256_bundle.crt.zip

artifacts/cabundle/COMODO_DV_SHA-256_bundle.crt: artifacts/cabundle/comodo_dv_sha-256_bundle.crt.zip
	unzip -o artifacts/cabundle/comodo_dv_sha-256_bundle.crt.zip -d "$(@D)"
	touch -c "$@"

artifacts/cacert.pem: artifacts/cabundle/gd_bundle-g2-g1.crt artifacts/cabundle/COMODO_DV_SHA-256_bundle.crt
	curl -L -o "$@.orig" http://curl.haxx.se/ca/cacert.pem
	cat "$@.orig" > "$@"
	( echo ""; echo "Go Daddy Intermediates"; echo "=================="; cat artifacts/cabundle/gd_bundle-g2-g1.crt ) >> "$@"
	( echo ""; echo "Comodo PositiveSSL Intermediates"; echo "=================="; cat artifacts/cabundle/COMODO_DV_SHA-256_bundle.crt ) >> "$@"

CURRENT_OS          := $(shell go run ./build/goos.go)
CURRENT_ARCH        := $(shell go run ./build/goarch.go)

BUILD_DEBUG_FLAGS   := -v
BUILD_RELEASE_FLAGS := -v -ldflags "-s -w"
BUILD_DEBUG_ENV     :=
BUILD_RELEASE_ENV   :=
BUILD_OS            := $(sort linux $(CURRENT_OS))
BUILD_ARCH          := $(sort amd64 $(CURRENT_ARCH))
BUILD_MATRIX        := $(foreach OS,$(BUILD_OS),$(foreach ARCH,$(BUILD_ARCH),$(OS)/$(ARCH)))

BUILD_PATH          := artifacts/build
COVERAGE_PATH       := artifacts/tests/coverage
BINARIES_PATH       := src/cmd

SOURCES             := $(shell find ./src -name *.go)
PACKAGES            := $(sort $(dir $(SOURCES)))
BINARIES            := $(notdir $(shell find $(BINARIES_PATH) -mindepth 1 -maxdepth 1 -type d))
TARGETS             := $(foreach BUILD,$(BUILD_MATRIX),$(foreach BIN,$(BINARIES),$(BUILD)/$(BIN)))

test: vendor
	go test $(PACKAGES)

build: $(addprefix $(BUILD_PATH)/debug/$(CURRENT_OS)/$(CURRENT_ARCH)/,$(BINARIES))

debug: $(addprefix $(BUILD_PATH)/debug/,$(TARGETS))

release: $(addprefix $(BUILD_PATH)/release/,$(TARGETS))

docker: artifacts/docker.touch

clean:
	@git check-ignore ./* | grep -v ^./vendor | xargs -t -n1 rm -rf

clean-all:
	@git check-ignore ./* | xargs -t -n1 rm -rf

coverage: $(COVERAGE_PATH)/index.html

open-coverage: $(COVERAGE_PATH)/index.html
	open $(COVERAGE_PATH)/index.html

lint: vendor
	go vet $(PACKAGES)
	! go fmt $(PACKAGES) | grep ^

prepare: lint coverage
	travis lint

ci: lint $(COVERAGE_PATH)/coverage.cov docker

.PHONY: build test debug release docker clean clean-all coverage open-coverage lint prepare ci

GLIDE := $(GOPATH)/bin/glide
$(GLIDE):
	go get -u github.com/Masterminds/glide

vendor: glide.lock | $(GLIDE)
	$(GLIDE) install
	@touch vendor

glide.lock: glide.yaml | $(GLIDE)
	$(GLIDE) update
	@touch vendor

$(BUILD_PATH)/%: vendor $(SOURCES)
	$(eval PARTS := $(subst /, ,$*))
	$(eval BUILD := $(word 1,$(PARTS)))
	$(eval OS    := $(word 2,$(PARTS)))
	$(eval ARCH  := $(word 3,$(PARTS)))
	$(eval PKG   := ./$(BINARIES_PATH)/$(word 4,$(PARTS)))
	$(eval FLAGS := $(if $(findstring debug,$(BUILD)),$(BUILD_DEBUG_FLAGS),$(BUILD_RELEASE_FLAGS)))
	$(eval ENV   := $(if $(findstring debug,$(BUILD)),$(BUILD_DEBUG_ENV),$(BUILD_RELEASE_ENV)))

	GOARCH=$(ARCH) GOOS=$(OS) $(ENV) go build $(FLAGS) -o "$@" "$(PKG)"

GOCOVMERGE := $(GOPATH)/bin/gocovmerge
$(GOCOVMERGE):
	go get -u github.com/wadey/gocovmerge

$(COVERAGE_PATH)/index.html: $(COVERAGE_PATH)/coverage.cov
	go tool cover -html="$<" -o "$@"

$(COVERAGE_PATH)/coverage.cov: $(foreach P,$(PACKAGES),$(COVERAGE_PATH)/$(P)coverage.partial) | $(GOCOVMERGE)
	@mkdir -p $(@D)
	$(GOCOVMERGE) $^ > "$@"

.SECONDEXPANSION:
%/coverage.partial: vendor $$(subst $(COVERAGE_PATH)/,,$$(@D))/*.go
	$(eval PKG := $(subst $(COVERAGE_PATH)/,,$*))
	@mkdir -p "$(@D)"
	@touch "$@"
	go test "$(PKG)" -covermode=count -coverprofile="$@"

artifacts/docker.touch: docker/Dockerfile $(BUILD_PATH)/release/linux/amd64/honeycomb $(shell find ./docker -type f)
	docker build -f "$<" -t honeycomb:dev .
	touch "$@"

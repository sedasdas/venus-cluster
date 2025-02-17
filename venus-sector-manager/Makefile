unexport GOTAGS
unexport GOFLAGS

FFI_PATH:=extern/filecoin-ffi/
FFI_DEPS:=.install-filcrypto
FFI_DEPS:=$(addprefix $(FFI_PATH),$(FFI_DEPS))

$(FFI_DEPS): build-dep/.filecoin-install ;

MODULES:=

CLEAN:=
BINS:=
BUILDS:=

build-dep/.filecoin-install: $(FFI_PATH)
	$(MAKE) -C $(FFI_PATH) $(FFI_DEPS:$(FFI_PATH)%=%)
	@touch $@

MODULES+=$(FFI_PATH)
BUILD_DEPS+=build-dep/.filecoin-install
CLEAN+=build-dep/.filecoin-install

$(MODULES): build-dep/.update-modules ;

# dummy file that marks the last time modules were updated
build-dep/.update-modules:
	git submodule update --init --recursive
	touch $@

CLEAN+=build-dep/.update-modules

clean:
	rm -rf $(CLEAN) $(BINS)
	-$(MAKE) -C $(FFI_PATH) clean
.PHONY: clean

ldflags=-X=github.com/ipfs-force-community/venus-cluster/venus-sector-manager/ver.Commit=git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"


build-venus-sector-manager:
	rm -rf ./venus-sector-manager
	go build -tags="$(GOTAGS)" $(GOFLAGS) -o venus-sector-manager ./cmd/venus-sector-manager

build-plugin-fsstore:
	rm -rf ./plugin-fsstore.so
	go build -tags="$(GOTAGS)" $(GOFLAGS) -buildmode=plugin -o plugin-fsstore.so ./plugin/fsstore/

.PHONY: venus-sector-manager
BINS+=venus-sector-manager
BUILDS+=build-venus-sector-manager
BUILDS+=build-plugin-fsstore

build-all: GOTAGS+=prod
build-all: $(BUILD_DEPS) $(BUILDS)

build-all-dev: $(MODULES) $(BUILDS)

test-all: GOTAGS+=prod
test-all: $(BUILD_DEPS)
	go test -tags="$(GOTAGS)" $(GOFLAGS) -v -failfast `go list ./... | grep -v /extern/`

check-all: check-lint check-go-mod

check-lint: $(BUILD_DEPS)
	golangci-lint run

check-go-mod:
	go mod tidy

dev-env:
	go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.1

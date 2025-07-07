
EXECUTABLE := qng
EXECUTABLE_QX := qx
GITVER := $(shell git rev-parse --short=7 HEAD )
GITDIRTY := $(shell git diff --quiet || echo '-dirty')
GITVERSION = "$(GITVER)$(GITDIRTY)"
ifeq ($(DEV),)
	DEV := dev
endif
RELEASE=release
LDFLAG_DEV = -X github.com/Qitmeer/qng/version.Build=$(DEV)-$(GITVERSION)
LDFLAG_RELEASE = -X github.com/Qitmeer/qng/version.Build=$(RELEASE)-$(GITVERSION)
GOFLAGS_DEV = -ldflags "$(LDFLAG_DEV)"
GOFLAGS_RELEASE = -ldflags "$(LDFLAG_RELEASE)"
GOFLAGS_RELEASE_QX = -ldflags "$(LDFLAG_RELEASE)"
VERSION=$(shell ./build/bin/qng --version | grep ^QNG | cut -d' ' -f3|cut -d'+' -f1)
GOBIN = ./build/bin
RELEASE_DIR = ./build/release
# Automatic detection operating system
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S), Linux)
	OUTPUT_SUFFIX=
	GOOS=linux
    GOARCH=amd64

else ifeq ($(UNAME_S), Darwin)
	OUTPUT_SUFFIX=
	# https://github.com/golang/go/issues/67799
    GOFLAGS_DEV = -ldflags="$(LDFLAG_DEV) -extldflags=-Wl,-no_warn_duplicate_libraries"
    GOFLAGS_RELEASE = -ldflags="$(LDFLAG_RELEASE) -extldflags=-Wl,-no_warn_duplicate_libraries"
    GOOS=darwin
    GOARCH=arm64

else ifeq ($(findstring MINGW,$(UNAME_S)), MINGW)
	OUTPUT_SUFFIX=.exe
	GOOS=windows
    GOARCH=amd64

else
	OUTPUT_SUFFIX=

endif

EXECUTABLE_PATH := $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/bin/$(EXECUTABLE)$(OUTPUT_SUFFIX)
EXECUTABLE_QX_PATH := $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/bin/$(EXECUTABLE_QX)


EXECUTABLES=$(EXECUTABLE_PATH) $(EXECUTABLE_QX_PATH)

DEV_EXECUTABLES := \
	build/dev/$(GOOS)/$(GOARCH)/bin/$(EXECUTABLE) \
	build/dev/$(GOOS)/$(GOARCH)/bin/$(EXECUTABLE).exe

COMPRESSED_EXECUTABLES := $(EXECUTABLE_PATH:%=%.qng.tar.gz) $(EXECUTABLE_QX_PATH:%=%.qx.tar.gz)


RELEASE_TARGETS=$(EXECUTABLES) $(COMPRESSED_EXECUTABLES)

DEV_TARGETS=$(DEV_EXECUTABLES)

ZMQ = FALSE

DEBUG = OFF

.PHONY: qng qx release

qng: qng-build
	@echo "Done building."
	@echo "  $(shell $(GOBIN)/qng --version))"
	@echo "Run \"$(GOBIN)/qng\" to launch."

qng-build:
    ifeq ($(ZMQ),TRUE)
		@echo "Enalbe ZMQ"
		@go build -o $(GOBIN)/qng$(OUTPUT_SUFFIX) $(GOFLAGS_DEV) -tags=zmq "github.com/Qitmeer/qng/cmd/qng"
    else ifeq ($(DEBUG),ON)
		@echo "Enable DEBUG"
		@go build -o $(GOBIN)/qng$(OUTPUT_SUFFIX) $(GOFLAGS_DEV) -gcflags="all=-N -l" "github.com/Qitmeer/qng/cmd/qng"
    else
		@go build -o $(GOBIN)/qng$(OUTPUT_SUFFIX) $(GOFLAGS_RELEASE) "github.com/Qitmeer/qng/cmd/qng"
    endif
qx:
	@go build -o $(GOBIN)/qx $(GOFLAGS_DEV) "github.com/Qitmeer/qng/cmd/qx"
relay:
	@go build -o $(GOBIN)/relaynode $(GOFLAGS_DEV) "github.com/Qitmeer/qng/cmd/relaynode"

generate-contracts-pkg:
	@git submodule update --init --recursive
	@if [ ! -d $(GOBIN) ]; then \
		echo "Directory $(GOBIN) does not exist. Creating it now..."; \
		mkdir -p $(GOBIN); \
	fi
	@if [ ! -e "$(GOBIN)/MeerChange.abi" ]; then \
        touch $(GOBIN)/MeerChange.abi; \
        jq -r '.abi' ./meerevm/contracts/meerchange/contracts/artifacts/MeerChange.json > $(GOBIN)/MeerChange.abi; \
        touch $(GOBIN)/MeerChange.bin; \
        jq -r '.bytecode' ./meerevm/contracts/meerchange/contracts/artifacts/MeerChange.json > $(GOBIN)/MeerChange.bin; \
        touch $(GOBIN)/EntryPoint.abi; \
        jq -r '.abi' ./meerevm/contracts/eip4337-account-abstraction/contracts/artifacts/EntryPoint.json > $(GOBIN)/EntryPoint.abi; \
        touch $(GOBIN)/QngAccount.abi; \
        jq -r '.abi' ./meerevm/contracts/eip4337-account-abstraction/contracts/artifacts/QngAccount.json > $(GOBIN)/QngAccount.abi; \
    fi
	@abigen --abi=$(GOBIN)/MeerChange.abi --bin=$(GOBIN)/MeerChange.bin --pkg=meerchange --out=./meerevm/meer/meerchange/meerchange.go
	@abigen --abi=$(GOBIN)/EntryPoint.abi --pkg=entrypoint --out=./meerevm/meer/entrypoint/entrypoint.go
	@abigen --abi=$(GOBIN)/QngAccount.abi --pkg=qngaccount --out=./meerevm/meer/qngaccount/qngaccount.go

checkversion: qng-build
#	@echo version $(VERSION)

all: qng-build qx relay

# amd64 release
build/release/%/$(EXECUTABLE):
	@echo Build $(@)
	go build $(GOFLAGS_RELEASE) -o $(@) "github.com/Qitmeer/qng/cmd/qng"
build/release/%/$(EXECUTABLE).exe:
	@echo Build $(@)
	go build $(GOFLAGS_RELEASE) -o $(@) "github.com/Qitmeer/qng/cmd/qng"
build/release/%/$(EXECUTABLE_QX):
	@echo Build $(@)
	go build $(GOFLAGS_RELEASE_QX) -o $(@) "github.com/Qitmeer/qng/cmd/qx"
build/release/%/$(EXECUTABLE_QX).exe:
	@echo Build $(@)
	go build $(GOFLAGS_RELEASE_QX) -o $(@) "github.com/Qitmeer/qng/cmd/qx"


# amd64 dev
build/dev/%: OS=$(word 3,$(subst /, ,$(@)))
build/dev/%: ARCH=$(word 4,$(subst /, ,$(@)))
build/dev/%/$(EXECUTABLE):
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_DEV) -o $(@) "github.com/Qitmeer/qng/cmd/qng"
build/dev/%/$(EXECUTABLE).exe:
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_DEV) -o $(@) "github.com/Qitmeer/qng/cmd/qng"


%.qng.zip: %.exe
	@echo zip $(EXECUTABLE)-$(VERSION)-$(GOOS)-$(ARCH)
	@zip $(RELEASE_DIR)/$(EXECUTABLE)-$(VERSION)-$(GOOS)-$(GOARCH).zip "$<"

%.qng.cn.zip: %.exe
	@echo zip $(EXECUTABLE)-$(VERSION)-$(GOOS)-$(ARCH)
	@zip -j $(RELEASE_DIR)/$(EXECUTABLE)-$(VERSION)-$(GOOS)-$(GOARCH).cn.zip "$<" script/win/start.bat

%.qng.tar.gz : %
	@echo tar $(EXECUTABLE)-$(VERSION)-$(GOOS)-$(ARCH)
	@tar -zcvf $(RELEASE_DIR)/$(EXECUTABLE)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz "$<"

%.qx.tar.gz : %
	@echo qx.tar.gz: $(@)
	@tar -zcvf $(RELEASE_DIR)/$(EXECUTABLE_QX)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz "$<"
%.qx.zip: %.exe
	@echo qx.zip: $(@)
	@echo zip $(EXECUTABLE_QX)-$(VERSION)-$(GOOS)-$(GOARCH)
	@zip $(RELEASE_DIR)/$(EXECUTABLE_QX)-$(VERSION)-$(GOOS)-$(GOARCH).zip "$<"

release: clean checkversion
	@echo "Build release version : $(VERSION)"
	@$(MAKE) $(RELEASE_TARGETS)

	@if [ "$(GOOS)" = "windows" ]; then \
		openssl sha512 $(EXECUTABLES) > $(RELEASE_DIR)/$(VERSION)-$(GOOS)-$(GOARCH)_checksum.txt; \
		openssl sha512 $(RELEASE_DIR)/$(EXECUTABLE)-$(VERSION)-* >> $(RELEASE_DIR)/$(VERSION)-$(GOOS)-$(GOARCH)_checksum.txt; \
		openssl sha512 $(RELEASE_DIR)/$(EXECUTABLE_QX)-$(VERSION)-* >> $(RELEASE_DIR)/$(VERSION)-$(GOOS)-$(GOARCH)_checksum.txt; \
	else \
		shasum -a 512 $(EXECUTABLES) > $(RELEASE_DIR)/$(VERSION)-$(GOOS)-$(GOARCH)_checksum.txt; \
		shasum -a 512 $(RELEASE_DIR)/$(EXECUTABLE)-$(VERSION)-* >> $(RELEASE_DIR)/$(VERSION)-$(GOOS)-$(GOARCH)_checksum.txt; \
		shasum -a 512 $(RELEASE_DIR)/$(EXECUTABLE_QX)-$(VERSION)-* >> $(RELEASE_DIR)/$(VERSION)-$(GOOS)-$(GOARCH)_checksum.txt; \
	fi

dev: clean checkversion
	@echo "Build dev version : $(VERSION)"
	@$(MAKE) $(DEV_TARGETS)

checksum: checkversion
	@cat $(RELEASE_DIR)/release-$(VERSION)_checksum.txt|shasum -c
clean:
	@rm -f *.zip
	@rm -f *.tar.gz
	@rm -f ./build/bin/*
	@rm -rf ./build/release
	@rm -rf ./build/dev

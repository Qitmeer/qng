
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

UNIX_EXECUTABLES := \
	build/release/darwin/amd64/bin/$(EXECUTABLE) \
	build/release/darwin/arm64/bin/$(EXECUTABLE) \
	build/release/linux/amd64/bin/$(EXECUTABLE)
WIN_EXECUTABLES := \
	build/release/windows/amd64/bin/$(EXECUTABLE).exe

UNIX_EXECUTABLES_QX := \
	build/release/darwin/amd64/bin/$(EXECUTABLE_QX) \
	build/release/darwin/arm64/bin/$(EXECUTABLE_QX) \
	build/release/linux/amd64/bin/$(EXECUTABLE_QX)
WIN_EXECUTABLES_QX := \
	build/release/windows/amd64/bin/$(EXECUTABLE_QX).exe


EXECUTABLES=$(UNIX_EXECUTABLES) $(WIN_EXECUTABLES) $(UNIX_EXECUTABLES_QX) $(WIN_EXECUTABLES_QX)

DEV_EXECUTABLES := \
	build/dev/darwin/amd64/bin/$(EXECUTABLE) \
	build/dev/darwin/arm64/bin/$(EXECUTABLE) \
	build/dev/linux/amd64/bin/$(EXECUTABLE) \
	build/dev/windows/amd64/bin/$(EXECUTABLE).exe

COMPRESSED_EXECUTABLES := \
     $(UNIX_EXECUTABLES:%=%.qng.tar.gz) \
     $(WIN_EXECUTABLES:%.exe=%.qng.zip) \
     $(WIN_EXECUTABLES:%.exe=%.qng.cn.zip) \
     $(UNIX_EXECUTABLES_QX:%=%.qx.tar.gz) \
     $(WIN_EXECUTABLES_QX:%.exe=%.qx.zip)


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
		@go build -o $(GOBIN)/qng $(GOFLAGS_DEV) -tags=zmq "github.com/Qitmeer/qng/cmd/qng"
    else ifeq ($(DEBUG),ON)
		@echo "Enable DEBUG"
		@go build -o $(GOBIN)/qng $(GOFLAGS_DEV) -gcflags="all=-N -l" "github.com/Qitmeer/qng/cmd/qng"
    else
		@go build -o $(GOBIN)/qng $(GOFLAGS_DEV) "github.com/Qitmeer/qng/cmd/qng"
    endif
qx:
	@go build -o $(GOBIN)/qx $(GOFLAGS_DEV) "github.com/Qitmeer/qng/cmd/qx"
relay:
	@go build -o $(GOBIN)/relaynode $(GOFLAGS_DEV) "github.com/Qitmeer/qng/cmd/relaynode"

checkversion: qng-build
#	@echo version $(VERSION)

all: qng-build qx relay

# amd64 release
build/release/%: OS=$(word 3,$(subst /, ,$(@)))
build/release/%: ARCH=$(word 4,$(subst /, ,$(@)))
build/release/%/$(EXECUTABLE):
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_RELEASE) -o $(@) "github.com/Qitmeer/qng/cmd/qng"
build/release/%/$(EXECUTABLE).exe:
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_RELEASE) -o $(@) "github.com/Qitmeer/qng/cmd/qng"
build/release/%/$(EXECUTABLE_QX):
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_RELEASE_QX) -o $(@) "github.com/Qitmeer/qng/cmd/qx"
build/release/%/$(EXECUTABLE_QX).exe:
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_RELEASE_QX) -o $(@) "github.com/Qitmeer/qng/cmd/qx"


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
	@echo zip $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH)
	@zip $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH).zip "$<"

%.qng.cn.zip: %.exe
	@echo zip $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH)
	@zip -j $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH).cn.zip "$<" script/win/start.bat

%.qng.tar.gz : %
	@echo tar $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH)
	@tar -zcvf $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH).tar.gz "$<"

%.qx.tar.gz : %
	@echo qx.tar.gz: $(@)
	@tar -zcvf $(EXECUTABLE_QX)-$(VERSION)-$(OS)-$(ARCH).tar.gz "$<"
%.qx.zip: %.exe
	@echo qx.zip: $(@)
	@echo zip $(EXECUTABLE_QX)-$(VERSION)-$(OS)-$(ARCH)
	@zip $(EXECUTABLE_QX)-$(VERSION)-$(OS)-$(ARCH).zip "$<"


release: clean checkversion
	@echo "Build release version : $(VERSION)"
	@$(MAKE) $(RELEASE_TARGETS)
	@shasum -a 512 $(EXECUTABLES) > release-$(VERSION)_checksum.txt
	@shasum -a 512 $(EXECUTABLE)-$(VERSION)-* >> release-$(VERSION)_checksum.txt
	@shasum -a 512 $(EXECUTABLE_QX)-$(VERSION)-* >> release-$(VERSION)_checksum.txt
dev: clean checkversion
	@echo "Build dev version : $(VERSION)"
	@$(MAKE) $(DEV_TARGETS)

checksum: checkversion
	@cat release-$(VERSION)_checksum.txt|shasum -c
clean:
	@rm -f *.zip
	@rm -f *.tar.gz
	@rm -f ./build/bin/*
	@rm -rf ./build/release
	@rm -rf ./build/dev

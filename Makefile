
EXECUTABLE := qng
GITVER := $(shell git rev-parse --short=7 HEAD )
GITDIRTY := $(shell git diff --quiet || echo '-dirty')
GITVERSION = "$(GITVER)$(GITDIRTY)"
DEV=dev
RELEASE=release
LDFLAG_DEV = -X github.com/Qitmeer/qng/version.Build=$(DEV)-$(GITVERSION)
LDFLAG_RELEASE = -X github.com/Qitmeer/qng/version.Build=$(RELEASE)-$(GITVERSION)
GOFLAGS_DEV = -ldflags "$(LDFLAG_DEV)"
GOFLAGS_RELEASE = -ldflags "$(LDFLAG_RELEASE)"
VERSION=$(shell ./build/bin/qng --version | grep ^qng | cut -d' ' -f3|cut -d'+' -f1)
GOBIN = ./build/bin

UNIX_EXECUTABLES := \
	build/release/darwin/amd64/bin/$(EXECUTABLE) \
	build/release/linux/amd64/bin/$(EXECUTABLE)
WIN_EXECUTABLES := \
	build/release/windows/amd64/bin/$(EXECUTABLE).exe

EXECUTABLES=$(UNIX_EXECUTABLES) $(WIN_EXECUTABLES)

DEV_EXECUTABLES := \
	build/dev/darwin/amd64/bin/$(EXECUTABLE) \
	build/dev/linux/amd64/bin/$(EXECUTABLE) \
	build/dev/windows/amd64/bin/$(EXECUTABLE).exe

COMPRESSED_EXECUTABLES=$(UNIX_EXECUTABLES:%=%.tar.gz) $(WIN_EXECUTABLES:%.exe=%.zip) $(WIN_EXECUTABLES:%.exe=%.cn.zip)

RELEASE_TARGETS=$(EXECUTABLES) $(COMPRESSED_EXECUTABLES)

DEV_TARGETS=$(DEV_EXECUTABLES)

ZMQ = FALSE

.PHONY: qng release

qng: qng-build
	@echo "Done building."
	@echo "  $(shell $(GOBIN)/qng --version))"
	@echo "Run \"$(GOBIN)/qng\" to launch."

qng-build:
    ifeq ($(ZMQ),TRUE)
		@echo "Enalbe ZMQ"
		@go build -o $(GOBIN)/qng $(GOFLAGS_DEV) -tags=zmq "github.com/Qitmeer/qng/cmd/qng"
    else
		@go build -o $(GOBIN)/qng $(GOFLAGS_DEV) "github.com/Qitmeer/qng/cmd/qng"
    endif

meerdag:
	@go build -o $(GOBIN)/plugin/meerdag "github.com/Qitmeer/qng/consensus/meerdag"

checkversion: qng-build
#	@echo version $(VERSION)

all: qng-build

# amd64 release
build/release/%: OS=$(word 3,$(subst /, ,$(@)))
build/release/%: ARCH=$(word 4,$(subst /, ,$(@)))
build/release/%/$(EXECUTABLE):
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_RELEASE) -o $(@) "github.com/Qitmeer/qng/cmd/qng"
build/release/%/$(EXECUTABLE).exe:
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_RELEASE) -o $(@) "github.com/Qitmeer/qng/cmd/qng"

# amd64 dev
build/dev/%: OS=$(word 3,$(subst /, ,$(@)))
build/dev/%: ARCH=$(word 4,$(subst /, ,$(@)))
build/dev/%/$(EXECUTABLE):
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_DEV) -o $(@) "github.com/Qitmeer/qng/cmd/qng"
build/dev/%/$(EXECUTABLE).exe:
	@echo Build $(@)
	@GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS_DEV) -o $(@) "github.com/Qitmeer/qng/cmd/qng"


%.zip: %.exe
	@echo zip $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH)
	@zip $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH).zip "$<"

%.cn.zip: %.exe
	@echo Build $(@).cn.zip
	@echo zip $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH)
	@zip -j $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH).cn.zip "$<" script/win/start.bat

%.tar.gz : %
	@echo tar $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH)
	@tar -zcvf $(EXECUTABLE)-$(VERSION)-$(OS)-$(ARCH).tar.gz "$<"
release: clean checkversion
	@echo "Build release version : $(VERSION)"
	@$(MAKE) $(RELEASE_TARGETS)
	@shasum -a 512 $(EXECUTABLES) > $(EXECUTABLE)-$(VERSION)_checksum.txt
	@shasum -a 512 $(EXECUTABLE)-$(VERSION)-* >> $(EXECUTABLE)-$(VERSION)_checksum.txt
dev: clean checkversion
	@echo "Build dev version : $(VERSION)"
	@$(MAKE) $(DEV_TARGETS)

checksum: checkversion
	@cat $(EXECUTABLE)-$(VERSION)_checksum.txt|shasum -c
clean:
	@rm -f *.zip
	@rm -f *.tar.gz
	@rm -f ./build/bin/qng
	@rm -rf ./build/release
	@rm -rf ./build/dev

# === This file is part of ALICE O² ===
#
#  Copyright 2017-2018 CERN and copyright holders of ALICE O².
#  Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
#
#  Based on Torus project Makefile <https://github.com/coreos/torus>
#
#  This program is free software: you can redistribute it and/or modify
#  it under the terms of the GNU General Public License as published by
#  the Free Software Foundation, either version 3 of the License, or
#  (at your option) any later version.
#
#  This program is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#  GNU General Public License for more details.
#
#  You should have received a copy of the GNU General Public License
#  along with this program.  If not, see <http://www.gnu.org/licenses/>.
#
#  In applying this license CERN does not waive the privileges and
#  immunities granted to it by virtue of its status as an
#  Intergovernmental Organization or submit itself to any jurisdiction.

include VERSION

BUILD := `git rev-parse --short HEAD`

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

HOST_GOOS=$(shell go env GOOS)
HOST_GOARCH=$(shell go env GOARCH)
CGO_LDFLAGS=CGO_LDFLAGS="$(ROOT_DIR)/vendor/infoLoggerForGo/infoLoggerForGo.a -lstdc++"
BUILD_FLAGS=$(CGO_LDFLAGS) $(BUILD_ENV_FLAGS)
REPOPATH = github.com/AliceO2Group/Control

VERBOSE_1 := -v
VERBOSE_2 := -v -x

WHAT := o2control-core o2control-executor coconut peanut
WHAT_o2control-core_BUILD_FLAGS=$(CGO_LDFLAGS) $(BUILD_ENV_FLAGS)
WHAT_o2control-executor_BUILD_FLAGS=$(CGO_LDFLAGS) $(BUILD_ENV_FLAGS)
WHAT_coconut_BUILD_FLAGS=$(CGO_LDFLAGS) $(BUILD_ENV_FLAGS)
WHAT_peanut_BUILD_FLAGS=$(CGO_LDFLAGS) $(BUILD_ENV_FLAGS)

INSTALL_WHAT:=$(patsubst %, install_%, $(WHAT))


GENERATE_DIRS := ./core ./executor ./coconut/cmd
SRC_DIRS := ./cmd/* ./core ./coconut ./executor ./common ./configuration ./occ/peanut

# Use linker flags to provide version/build settings to the target
PROD :=-X=$(REPOPATH)/common/product
LDFLAGS=-ldflags "$(PROD).VERSION_MAJOR=$(VERSION_MAJOR) $(PROD).VERSION_MINOR=$(VERSION_MINOR) $(PROD).VERSION_PATCH=$(VERSION_PATCH) $(PROD).BUILD=$(BUILD)"
HAS_GOGOPROTO := $(shell command -v protoc-gen-gofast 2> /dev/null)

GO_GET_U1 := $(addprefix github.com/gogo/protobuf/, proto protoc-gen-gofast protoc-gen-gogofast protoc-gen-gogofaster protoc-gen-gogoslick gogoproto)
GO_GET_U2 := $(addprefix github.com/golang/protobuf/, proto protoc-gen-go)
GO_GET_U2 += google.golang.org/grpc

.PHONY: build all install generate test debugtest vet fmt clean cleanall help $(WHAT) tools vendor

build: $(WHAT)

all: vendor generate build

install: $(INSTALL_WHAT)
#	@for w in $(WHAT); do \
#	    FLAGS="WHAT_$${w}_BUILD_FLAGS"; \
#	    echo -e "${$${FLAGS}}"; \
#		echo -e "\e[1;33mgo install\e[0m ./cmd/$$w  \e[1;33m==>\e[0m  \e[1;34m$$GOPATH/bin/$$w\e[0m"; \
#		$(WHAT_$${w}_BUILD_FLAGS) go install $(VERBOSE_$(V)) $(LDFLAGS) ./cmd/$$w; \
#	done

$(WHAT):
#	@echo -e "WHAT_$@_BUILD_FLAGS $(WHAT_$@_BUILD_FLAGS)"
	@echo -e "\e[1;33m$(WHAT_$@_BUILD_FLAGS) go build\e[0m ./cmd/$@  \e[1;33m==>\e[0m  \e[1;34m./bin/$@\e[0m"
#	@echo ${PWD}
	@$(WHAT_$@_BUILD_FLAGS) go build $(VERBOSE_$(V)) -o bin/$@ $(LDFLAGS) ./cmd/$@

$(INSTALL_WHAT):
#	@echo -e "WHAT_$(@:install_%=%)_BUILD_FLAGS $(WHAT_$(@:install_%=%)_BUILD_FLAGS)"
	@echo -e "\e[1;33m$(WHAT_$(@:install_%=%)_BUILD_FLAGS) go install\e[0m ./cmd/$(@:install_%=%)  \e[1;33m==>\e[0m  \e[1;34m$$GOPATH/bin/$(@:install_%=%)\e[0m"
#	@echo ${PWD}
	@$(WHAT_$(@:install_%=%)_BUILD_FLAGS) go install $(VERBOSE_$(V)) $(LDFLAGS) ./cmd/$(@:install_%=%)

generate:
ifndef HAS_GOGOPROTO
	$(MAKE) tools/protoc
endif
	@for gendir in $(GENERATE_DIRS); do \
		echo -e "\e[1;33mgo generate\e[0m $$gendir"; \
		go generate $(VERBOSE_$(V)) $$gendir; \
	done

test: tools/dep
	go test -v --race $(SRC_DIRS) -ginkgo.progress

debugtest: tools/dep
	go test -v --race $(SRC_DIRS) -ginkgo.v -ginkgo.trace -ginkgo.progress

vet: tools/dep
	go vet $(SRC_DIRS)

fmt: tools/dep
	go fmt $(SRC_DIRS)

clean:
	@rm -rf ./bin/*
	@echo -e "clean done: \e[1;34mbin\e[0m"

cleanall:
	@rm -rf bin tools vendor
	@echo -e "clean done: \e[1;34mbin tools vendor\e[0m"

vendor: tools/dep
	@echo -e "\e[1;33mdep ensure\e[0m"
	@./tools/dep ensure
	@mkdir -p vendor/infoLoggerForGo
	@cp ${INFOLOGGER_ROOT}/lib/infoLoggerForGo.* vendor/infoLoggerForGo/

tools: tools/dep tools/protoc

tools/dep:
	@echo "downloading dep"
	mkdir -p tools
	curl -L https://github.com/golang/dep/releases/download/v0.5.0/dep-$(HOST_GOOS)-$(HOST_GOARCH) -o tools/dep
	chmod +x tools/dep

tools/protoc:
	@echo "installing Go protoc"
	go get -u $(GO_GET_U1)
	go get -u $(GO_GET_U2)

help:
	@echo "available make variables:"
	@echo "  V                 - Build verbosity {0,1,2}."
	@echo "  BUILD_ENV_FLAGS   - Environment added to 'go build'."
	@echo "  WHAT              - Command to build. (e.g. WHAT=o2control-core)"

# === This file is part of ALICE O² ===
#
#  Copyright 2017-2020 CERN and copyright holders of ALICE O².
#  Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
#
#  Partially derived from Torus Makefile <https://github.com/coreos/torus>
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
CGO_LDFLAGS=
BUILD_FLAGS=$(BUILD_ENV_FLAGS)
ifdef WITH_LIBINFOLOGGER
CGO_LDFLAGS=CGO_LDFLAGS="$(ROOT_DIR)/vendor/infoLoggerForGo/infoLoggerForGo.a -static-libstdc++ -static-libgcc"
BUILD_FLAGS=$(CGO_LDFLAGS) $(BUILD_ENV_FLAGS)
endif
REPOPATH = github.com/AliceO2Group/Control
ODC_PROTO="https://raw.githubusercontent.com/FairRootGroup/ODC/master/grpc-proto/odc.proto"

VERBOSE_1 := -v
VERBOSE_2 := -v -x

WHAT := o2-aliecs-core o2-aliecs-executor coconut peanut o2-aliecs-odc-shim walnut o2-apricot
WHAT_o2-aliecs-core_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_o2-aliecs-executor_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_coconut_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_peanut_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_o2-aliecs-odc-shim_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_walnut_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_o2-apricot_BUILD_FLAGS=$(BUILD_ENV_FLAGS)

INSTALL_WHAT:=$(patsubst %, install_%, $(WHAT))


GENERATE_DIRS := ./apricot ./coconut/cmd ./core ./executor ./core/integration/dcs ./odcshim ./walnut
SRC_DIRS := ./apricot ./cmd/* ./core ./coconut ./executor ./common ./configuration ./occ/peanut ./odcshim ./walnut

# Use linker flags to provide version/build settings to the target
PROD :=-X=$(REPOPATH)/common/product
EXTLDFLAGS :="-static"
LDFLAGS=-ldflags "-extldflags $(EXTLDFLAGS) $(PROD).VERSION_MAJOR=$(VERSION_MAJOR) $(PROD).VERSION_MINOR=$(VERSION_MINOR) $(PROD).VERSION_PATCH=$(VERSION_PATCH) $(PROD).BUILD=$(BUILD)" -tags osusergo,netgo

# We expect to find the protoc-gen-go executable in $GOPATH/bin
GOPATH := $(shell go env GOPATH)
GOPROTOCPATH=$(ROOT_DIR)/tools/protoc-gen-go
HAS_PROTOC := $(shell command -v $(GOPROTOCPATH) 2> /dev/null)

.PHONY: build all install generate test debugtest vet fmt clean cleanall help $(WHAT) tools vendor doc docs

build: $(WHAT)

all: vendor generate build

install: $(INSTALL_WHAT)
#	@for w in $(WHAT); do \
#	    FLAGS="WHAT_$${w}_BUILD_FLAGS"; \
#	    echo -e "${$${FLAGS}}"; \
#		echo -e "\033[1;33mgo install\033[0m ./cmd/$$w  \033[1;33m==>\033[0m  \033[1;34m$$GOPATH/bin/$$w\033[0m"; \
#		$(WHAT_$${w}_BUILD_FLAGS) go install $(VERBOSE_$(V)) $(LDFLAGS) ./cmd/$$w; \
#	done

$(WHAT):
#	@echo -e "WHAT_$@_BUILD_FLAGS $(WHAT_$@_BUILD_FLAGS)"
	@echo -e "\033[1;33mgo build -mod=vendor\033[0m ./cmd/$@  \033[1;33m==>\033[0m  \033[1;34m./bin/$@\033[0m"
#	@echo ${PWD}
	@$(WHAT_$@_BUILD_FLAGS) go build -mod=vendor $(VERBOSE_$(V)) -o bin/$@ $(LDFLAGS) ./cmd/$@

$(INSTALL_WHAT):
#	@echo -e "WHAT_$(@:install_%=%)_BUILD_FLAGS $(WHAT_$(@:install_%=%)_BUILD_FLAGS)"
	@echo -e "\033[1;33mgo install -mod=vendor\033[0m ./cmd/$(@:install_%=%)  \033[1;33m==>\033[0m  \033[1;34m$$GOPATH/bin/$(@:install_%=%)\033[0m"
#	@echo ${PWD}
	@$(WHAT_$(@:install_%=%)_BUILD_FLAGS) go install -mod=vendor $(VERBOSE_$(V)) $(LDFLAGS) ./cmd/$(@:install_%=%)

generate:
ifndef HAS_PROTOC
	$(MAKE) tools/protoc
endif
	@for gendir in $(GENERATE_DIRS); do \
		echo -e "\033[1;33mgo generate\033[0m $$gendir"; \
		PATH="$(ROOT_DIR)/tools:$$PATH" go generate $(VERBOSE_$(V)) $$gendir; \
	done

test:
	$(BUILD_FLAGS) go test -v --race $(SRC_DIRS) -ginkgo.progress

debugtest:
	$(BUILD_FLAGS) go test -v --race $(SRC_DIRS) -ginkgo.v -ginkgo.trace -ginkgo.progress

vet:
	go vet $(SRC_DIRS)

fmt:
	go fmt $(SRC_DIRS)

clean:
	@rm -rf ./bin/*
	@echo -e "clean done: \033[1;34mbin\033[0m"

cleanall:
	@rm -rf bin tools vendor
	@echo -e "clean done: \033[1;34mbin tools vendor\033[0m"

vendor:
	@echo -e "\033[1;33mgo mod vendor\033[0m"
	@go mod vendor

	@echo -e "\033[1;33mcurl odc.proto\033[0m"
	@mkdir -p odcshim/odcprotos
	@curl -s -L $(ODC_PROTO) -o odcshim/odcprotos/odc.proto

# WORKAROUND: In order to avoid the following issues:
# https://github.com/golang/protobuf/issues/992
# https://github.com/golang/protobuf/issues/1158
# we insert a go_package specification into the ODC protofile right
# after we download it.
	@echo -e "\033[1;33mpatch odc.proto\033[0m"
	@perl -pi -e '$$_.="option go_package = \"odcprotos;odc\";\n" if (/^package/)' odcshim/odcprotos/odc.proto

# vendor: tools/dep
#	@echo -e "\033[1;33mdep ensure\033[0m"
#	@./tools/dep ensure

#	@mkdir -p vendor/infoLoggerForGo
#	@cp ${INFOLOGGER_ROOT}/lib/infoLoggerForGo.* vendor/infoLoggerForGo/
# For cgo, *.a means C, so by default it will use gcc when linking against it. For
# this reason, we must create a dummy *.cpp file in the package dir to force cgo to
# use g++ instead of gcc.
#	@touch vendor/infoLoggerForGo/infoLoggerForGo.cpp

tools: tools/protoc

tools/protoc:
	@echo "installing Protobuf tools"

	@export GOBIN="$(ROOT_DIR)/tools" && cat common/tools/tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install -mod=readonly %

docs: doc

doc:
	@echo -e "generating coconut documentation  \033[1;33m==>\033[0m  \033[1;34m./coconut/doc\033[0m"
	@cd coconut/doc && go run . && cd ../..
	@echo -e "generating gRPC API documentation  \033[1;33m==>\033[0m  \033[1;34m./docs\033[0m"
	@cd apricot/protos && PATH="$(ROOT_DIR)/tools:$$PATH" protoc --doc_out="$(ROOT_DIR)/docs" --doc_opt=markdown,apidocs_apricot.md "apricot.proto"
	@cd core/protos && PATH="$(ROOT_DIR)/tools:$$PATH" protoc --doc_out="$(ROOT_DIR)/docs" --doc_opt=markdown,apidocs_aliecs.md "o2control.proto"
	@cd occ/protos && PATH="$(ROOT_DIR)/tools:$$PATH" protoc --doc_out="$(ROOT_DIR)/docs" --doc_opt=markdown,apidocs_occ.md "occ.proto"

help:
	@echo "available make variables:"
	@echo "  V                 - Build verbosity {0,1,2}."
	@echo "  BUILD_ENV_FLAGS   - Environment added to 'go build'."
	@echo "  WHAT              - Command to build. (e.g. WHAT=o2-aliecs-core)"

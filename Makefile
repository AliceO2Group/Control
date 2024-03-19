# === This file is part of ALICE O² ===
#
#  Copyright 2017-2023 CERN and copyright holders of ALICE O².
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

MINIMUM_SUPPORTED_GO_MAJOR_VERSION = 1
MINIMUM_SUPPORTED_GO_MINOR_VERSION = 20

GO_MAJOR_VERSION = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_MINOR_VERSION = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
WHICH_GO = $(shell which go)
GO_VERSION_VALIDATION_ERR_MSG = Go version $(GO_MAJOR_VERSION).$(GO_MINOR_VERSION) at $(WHICH_GO) is not supported, please update to at least $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION).$(MINIMUM_SUPPORTED_GO_MINOR_VERSION)

HOST_GOOS=$(shell go env GOOS)
HOST_GOARCH=$(shell go env GOARCH)
CGO_LDFLAGS=
BUILD_FLAGS=$(BUILD_ENV_FLAGS)
ifdef WITH_LIBINFOLOGGER
CGO_LDFLAGS=CGO_LDFLAGS="$(ROOT_DIR)/vendor/infoLoggerForGo/infoLoggerForGo.a -static-libstdc++ -static-libgcc"
BUILD_FLAGS=$(CGO_LDFLAGS) $(BUILD_ENV_FLAGS)
endif
REPOPATH = github.com/AliceO2Group/Control
DCS_PROTO="https://gitlab.cern.ch/api/v4/projects/137621/repository/files/proto%2Fdcs.proto/raw?ref=master"
ODC_PROTO="https://raw.githubusercontent.com/FairRootGroup/ODC/master/odc/grpc/odc.proto"
DD_PROTO="https://raw.githubusercontent.com/AliceO2Group/DataDistribution/master/src/DataDistControl/DataDistControl.proto"
BK_COM_PROTO="https://raw.githubusercontent.com/AliceO2Group/Bookkeeping/main/proto/common.proto"
BK_ENV_PROTO="https://raw.githubusercontent.com/AliceO2Group/Bookkeeping/main/proto/environment.proto"
BK_FLP_PROTO="https://raw.githubusercontent.com/AliceO2Group/Bookkeeping/main/proto/flp.proto"
BK_LOG_PROTO="https://raw.githubusercontent.com/AliceO2Group/Bookkeeping/main/proto/log.proto"
BK_RUN_PROTO="https://raw.githubusercontent.com/AliceO2Group/Bookkeeping/main/proto/run.proto"
BK_LHCFILL_PROTO="https://raw.githubusercontent.com/AliceO2Group/Bookkeeping/main/proto/lhcFill.proto"

VERBOSE_1 := -v
VERBOSE_2 := -v -x

WHAT := o2-aliecs-core o2-aliecs-executor coconut peanut walnut o2-apricot
WHAT_o2-aliecs-core_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_o2-aliecs-executor_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_coconut_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_peanut_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_walnut_BUILD_FLAGS=$(BUILD_ENV_FLAGS)
WHAT_o2-apricot_BUILD_FLAGS=$(BUILD_ENV_FLAGS)

INSTALL_WHAT:=$(patsubst %, install_%, $(WHAT))

GENERATE_DIRS := ./apricot ./coconut/cmd ./common ./common/runtype ./common/system ./core ./core/integration/ccdb ./core/integration/dcs ./core/integration/ddsched ./core/integration/kafka ./core/integration/odc ./executor ./walnut ./core/integration/trg ./core/integration/bookkeeping
SRC_DIRS := ./apricot ./cmd/* ./core ./coconut ./executor ./common ./configuration ./occ/peanut ./walnut
TEST_DIRS := ./configuration/cfgbackend ./configuration/componentcfg
GO_TEST_DIRS := ./core/repos ./core/integration/dcs

# Use linker flags to provide version/build settings to the target
PROD :=-X=$(REPOPATH)/common/product
EXTLDFLAGS :="-static"
LDFLAGS=-ldflags "-extldflags $(EXTLDFLAGS) $(PROD).VERSION_MAJOR=$(VERSION_MAJOR) $(PROD).VERSION_MINOR=$(VERSION_MINOR) $(PROD).VERSION_PATCH=$(VERSION_PATCH) $(PROD).BUILD=$(BUILD)" -tags osusergo,netgo

# We expect to find the protoc-gen-go executable in $GOPATH/bin
GOPATH := $(shell go env GOPATH)
GOPROTOCPATH=$(ROOT_DIR)/tools/protoc-gen-go
HAS_PROTOC := $(shell command -v $(GOPROTOCPATH) 2> /dev/null)

.EXPORT_ALL_VARIABLES:
CGO_ENABLED = 0

.PHONY: build all install generate test debugtest vet fmt clean cleanall help $(WHAT) tools vendor doc docs fdset

build: $(WHAT)

all: vendor generate build

validate-go-version: ## Validates the installed version of go against AliECS minimum requirement.
	@if [ $(GO_MAJOR_VERSION) -gt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		exit 0 ;\
	elif [ $(GO_MAJOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	elif [ $(GO_MINOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MINOR_VERSION) ] ; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	fi

install: $(INSTALL_WHAT)
#	@for w in $(WHAT); do \
#	    FLAGS="WHAT_$${w}_BUILD_FLAGS"; \
#	    echo -e "${$${FLAGS}}"; \
#		echo -e "\033[1;33mgo install\033[0m ./cmd/$$w  \033[1;33m==>\033[0m  \033[1;34m$$GOPATH/bin/$$w\033[0m"; \
#		$(WHAT_$${w}_BUILD_FLAGS) go install $(VERBOSE_$(V)) $(LDFLAGS) ./cmd/$$w; \
#	done

$(WHAT): validate-go-version
#	@echo -e "WHAT_$@_BUILD_FLAGS $(WHAT_$@_BUILD_FLAGS)"
	@echo -e "\033[1;33mgo build -mod=vendor\033[0m ./cmd/$@  \033[1;33m==>\033[0m  \033[1;34m./bin/$@\033[0m"
#	@echo ${PWD}
	@$(WHAT_$@_BUILD_FLAGS) go build -mod=vendor $(VERBOSE_$(V)) -o bin/$@ $(LDFLAGS) ./cmd/$@

# special case: if the current WHAT is o2-aliecs-executor, also copy over the shmcleaner script
	@if [ $@ == "o2-aliecs-executor" ]; then \
		echo -e "\033[1;33mcopy\033[0m ./o2-aliecs-shmcleaner  \033[1;33m==>\033[0m  \033[1;34m./bin/o2-aliecs-shmcleaner\033[0m"; \
		cp o2-aliecs-shmcleaner bin/o2-aliecs-shmcleaner; \
		chmod +x bin/o2-aliecs-shmcleaner; \
	fi; \

$(INSTALL_WHAT): validate-go-version
#	@echo -e "WHAT_$(@:install_%=%)_BUILD_FLAGS $(WHAT_$(@:install_%=%)_BUILD_FLAGS)"
	@echo -e "\033[1;33mgo install -mod=vendor\033[0m ./cmd/$(@:install_%=%)  \033[1;33m==>\033[0m  \033[1;34m$$GOPATH/bin/$(@:install_%=%)\033[0m"
#	@echo ${PWD}
	@$(WHAT_$(@:install_%=%)_BUILD_FLAGS) go install -mod=vendor $(VERBOSE_$(V)) $(LDFLAGS) ./cmd/$(@:install_%=%)

# special case: if the current WHAT is o2-aliecs-executor, also copy over the shmcleaner script
	@if [ $@ == "install_o2-aliecs-executor" ]; then \
		echo -e "\033[1;33minstall\033[0m ./o2-aliecs-shmcleaner  \033[1;33m==>\033[0m  \033[1;34m$$GOPATH/bin/o2-aliecs-shmcleaner\033[0m"; \
		cp o2-aliecs-shmcleaner $${GOPATH}/bin/o2-aliecs-shmcleaner; \
		chmod +x $${GOPATH}/bin/o2-aliecs-shmcleaner; \
	fi; \

generate:
ifndef HAS_PROTOC
	$(MAKE) tools/protoc
endif
	@for gendir in $(GENERATE_DIRS); do \
		echo -e "\033[1;33mgo generate\033[0m $$gendir"; \
		PATH="$(ROOT_DIR)/tools:$$PATH" go generate $(VERBOSE_$(V)) $$gendir; \
	done

test:
	@echo -e "[Ginkgo] \033[1;33mgo test -v\033[0m $(TEST_DIRS)\033[0m"
	@$(BUILD_FLAGS) go test -v $(TEST_DIRS) -ginkgo.show-node-events

	@echo -e "\n[gotest] \033[1;33mgo test -v\033[0m $(GO_TEST_DIRS)\033[0m"
	@$(BUILD_FLAGS) go test -v $(GO_TEST_DIRS)

debugtest:
	@echo -e "[Ginkgo] \033[1;33mgo test -v\033[0m $(TEST_DIRS)\033[0m"
	@$(BUILD_FLAGS) go test -v $(TEST_DIRS) -ginkgo.v -ginkgo.trace -ginkgo.show-node-events

	@echo -e "\n[gotest] \033[1;33mgo test -v\033[0m $(GO_TEST_DIRS)\033[0m"
	@$(BUILD_FLAGS) go test -v $(GO_TEST_DIRS)

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

vendor: validate-go-version
	@echo -e "\033[1;33mgo mod vendor\033[0m"
	@go mod vendor

# curl --header "PRIVATE-TOKEN: <your_access_token>" "https://gitlab.example.com/api/v4/projects/13083/repository/files/scripts%2Frun.sh/raw?ref=master"
	@echo -e "\033[1;33mcurl dcs.proto\033[0m"
	@mkdir -p core/integration/dcs/protos
	@curl --header "PRIVATE-TOKEN: glpat-z-r8s2Mfte8GBpLn7vcE" -s -L $(DCS_PROTO) -o core/integration/dcs/protos/dcs.proto

# FIXME: these two protofiles should be committed in order to be available offline
	@echo -e "\033[1;33mcurl odc.proto\033[0m"
	@mkdir -p core/integration/odc/protos
	@curl -s -L $(ODC_PROTO) -o core/integration/odc/protos/odc.proto

	@echo -e "\033[1;33mcurl ddsched.proto\033[0m"
	@mkdir -p core/integration/ddsched/protos
	@curl -s -L $(DD_PROTO) -o core/integration/ddsched/protos/ddsched.proto

	@echo -e "\033[1;33mcurl common.proto\033[0m"
	@mkdir -p core/integration/bookkeeping/protos
	@curl -s -L $(BK_COM_PROTO) -o core/integration/bookkeeping/protos/common.proto

	@echo -e "\033[1;33mcurl environment.proto\033[0m"
	@mkdir -p core/integration/bookkeeping/protos
	@curl -s -L $(BK_ENV_PROTO) -o core/integration/bookkeeping/protos/environment.proto

	@echo -e "\033[1;33mcurl flp.proto\033[0m"
	@mkdir -p core/integration/bookkeeping/protos
	@curl -s -L $(BK_FLP_PROTO) -o core/integration/bookkeeping/protos/flp.proto

	@echo -e "\033[1;33mcurl log.proto\033[0m"
	@mkdir -p core/integration/bookkeeping/protos
	@curl -s -L $(BK_LOG_PROTO) -o core/integration/bookkeeping/protos/log.proto

	@echo -e "\033[1;33mcurl run.proto\033[0m"
	@mkdir -p core/integration/bookkeeping/protos
	@curl -s -L $(BK_RUN_PROTO) -o core/integration/bookkeeping/protos/run.proto

	@echo -e "\033[1;33mcurl lhcFill.proto\033[0m"
	@mkdir -p core/integration/bookkeeping/protos
	@curl -s -L $(BK_LHCFILL_PROTO) -o core/integration/bookkeeping/protos/lhcFill.proto

# WORKAROUND: In order to avoid the following issues:
# https://github.com/golang/protobuf/issues/992
# https://github.com/golang/protobuf/issues/1158
# we insert a go_package specification into the ODC protofile right
# after we download it.
	@echo -e "\033[1;33mpatch odc.proto\033[0m"
	@perl -pi -e '$$_.="option go_package = \"github.com/AliceO2Group/Control/core/integration/odc/protos;odc\";\n" if (/^package/)' core/integration/odc/protos/odc.proto

	@echo -e "\033[1;33mpatch ddsched.proto\033[0m"
	@perl -pi -e '$$_.="option go_package = \"github.com/AliceO2Group/Control/core/integration/ddsched/protos;ddpb\";\n" if (/^package/)' core/integration/ddsched/protos/ddsched.proto

	@echo -e "\033[1;33mpatch common.proto\033[0m"
	@perl -pi -e '$$_.="option go_package = \"github.com/AliceO2Group/Control/core/integration/bookkeeping/protos;bkpb\";\n" if (/^package/)' core/integration/bookkeeping/protos/common.proto

	@echo -e "\033[1;33mpatch environment.proto\033[0m"
	@perl -pi -e '$$_.="option go_package = \"github.com/AliceO2Group/Control/core/integration/bookkeeping/protos;bkpb\";\n" if (/^package/)' core/integration/bookkeeping/protos/environment.proto

	@echo -e "\033[1;33mpatch flp.proto\033[0m"
	@perl -pi -e '$$_.="option go_package = \"github.com/AliceO2Group/Control/core/integration/bookkeeping/protos;bkpb\";\n" if (/^package/)' core/integration/bookkeeping/protos/flp.proto

	@echo -e "\033[1;33mpatch log.proto\033[0m"
	@perl -pi -e '$$_.="option go_package = \"github.com/AliceO2Group/Control/core/integration/bookkeeping/protos;bkpb\";\n" if (/^package/)' core/integration/bookkeeping/protos/log.proto

	@echo -e "\033[1;33mpatch run.proto\033[0m"
	@perl -pi -e '$$_.="option go_package = \"github.com/AliceO2Group/Control/core/integration/bookkeeping/protos;bkpb\";\n" if (/^package/)' core/integration/bookkeeping/protos/run.proto

	@echo -e "\033[1;33mpatch environment.proto\033[0m"
	@perl -pi -e 's/.*/import \"protos\/common\.proto\";/ if (/^import/)' core/integration/bookkeeping/protos/environment.proto

	@echo -e "\033[1;33mpatch flp.proto\033[0m"
	@perl -pi -e 's/.*/import \"protos\/common\.proto\";/ if (/^import/)' core/integration/bookkeeping/protos/flp.proto

	@echo -e "\033[1;33mpatch log.proto\033[0m"
	@perl -pi -e 's/.*/import \"protos\/common\.proto\";/ if (/^import/)' core/integration/bookkeeping/protos/log.proto

	@echo -e "\033[1;33mpatch run.proto\033[0m"
	@perl -pi -e 's/.*/import \"protos\/common\.proto\";/ if (/^import/)' core/integration/bookkeeping/protos/run.proto

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

docs: docs/coconut docs/grpc docs/swaggo

fdset:
	@echo -e "building fdset files  \033[1;33m==>\033[0m  \033[1;34m./common/protos\033[0m"

	@mkdir -p fdset
	@cd common/protos && protoc -o events.fdset events.proto && cd ../..
	@mv common/protos/events.fdset fdset

	@echo -e "to consume with \033[1;33mhttps://github.com/sevagh/pq\033[0m:  FDSET_PATH=./fdset pq kafka aliecs.environment --brokers kafka-broker-hostname:9092 --beginning --msgtype events.Event"

docs/coconut:
	@echo -e "generating coconut documentation  \033[1;33m==>\033[0m  \033[1;34m./coconut/doc\033[0m"
	@cd coconut/doc && go run . && cd ../..

docs/grpc:
	@echo -e "generating gRPC API documentation  \033[1;33m==>\033[0m  \033[1;34m./docs\033[0m"
	@cd apricot/protos && PATH="$(ROOT_DIR)/tools:$$PATH" protoc --doc_out="$(ROOT_DIR)/docs" --doc_opt=markdown,apidocs_apricot.md "apricot.proto"
	@cd core/protos && PATH="$(ROOT_DIR)/tools:$$PATH" protoc --doc_out="$(ROOT_DIR)/docs" --doc_opt=markdown,apidocs_aliecs.md "o2control.proto"
	@cd occ/protos && PATH="$(ROOT_DIR)/tools:$$PATH" protoc --doc_out="$(ROOT_DIR)/docs" --doc_opt=markdown,apidocs_occ.md "occ.proto"

docs/swaggo:
	@echo -e "generating REST API documentation  \033[1;33m==>\033[0m  \033[1;34m./apricot/docs\033[0m"
	@tools/swag fmt -d apricot
	@tools/swag init -o apricot/docs -d apricot/local,apricot,cmd/o2-apricot -g servicehttp.go

help:
	@echo "available make variables:"
	@echo "  V                 - Build verbosity {0,1,2}."
	@echo "  BUILD_ENV_FLAGS   - Environment added to 'go build'."
	@echo "  WHAT              - Command to build. (e.g. WHAT=o2-aliecs-core)"

# Copyright 2018 Canonical Ltd. 
# Licensed under the AGPLv3, see LICENCE file for details. 

# Makefile for monitoring utilities
#
PROJECT := github.com/cloud-green/monitoring
PROJECT_DIR := $(shell go list -e -f '{{.Dir}}' $(PROJECT))

ifneq ($(CURDIR),$(PROJECT_DIR))
$(error "$(CURDIR) not on GOPATH")
endif

ifndef GOBIN
GOBIN := $(shell mkdir -p $(GOPATH)/bin; realpath $(GOPATH))/bin
else
REAL_GOBIN := $(shell mkdir -p $(GOBIN); realpath $(GOBIN))
GOBIN := $(REAL_GOBIN)
endif

.PHONY: build
build: deps
	GOBIN=$(GOBIN) go build -a $(PROJECT)/...

.PHONY: check
check: build
	GOBIN=$(GOBIN) ./postgres.sh go test $(PROJECT)/...

.PHONY: race
race: build
	GOBIN=$(GOBIN) ./postgres.sh go test -race $(PROJECT)/...

.PHONY: land
land: check race

.PHONY: godeps
godeps: $(GOBIN)/godeps        
	GOBIN=$(GOBIN) $(GOBIN)/godeps -u dependencies.tsv

$(GOBIN)/godeps: $(GOBIN)      
	GOBIN=$(GOBIN) go get github.com/rogpeppe/godeps

ifeq ($(MAKE_GODEPS),true)
.PHONY: deps
deps: godeps
else
.PHONY: deps
deps:
	@echo "Skipping godeps. export MAKE_GODEPS = true to enable."
endif

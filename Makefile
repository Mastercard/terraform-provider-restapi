# Change these variables as necessary.
MAIN_PACKAGE_PATH := ./
BINARY_NAME := terraform-provider-restapi
GO := go
GO_VERSION ?= 1.21

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## test: run all tests
.PHONY: test
test:
	bash ./scripts/set-local-testing.rc
	bash ./scripts/test.sh

## build: build the application
.PHONY: build
build:
	# Include additional build steps compilation here...
	$(GO) build -o=${BINARY_NAME}.o ${MAIN_PACKAGE_PATH}

.PHONY: clean
clean :
	rm ${BINARY_NAME}.o

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
PACKAGE_PATH=./graph

all: test

# Run the tests
test:
	$(GOTEST) -v $(PACKAGE_PATH)

# Format the code
fmt:
	$(GOFMT) $(PACKAGE_PATH)/*.go

.PHONY: all test fmt

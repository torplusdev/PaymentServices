# Needed tools are installed to BINDIR.
#
# TODO: We share some tools across repos, so having this be local is not ideal.
# We should figure out a strategy for this BINDIR, but until then having a local
# folder helps us not depend on GOPATH at all (beyond what the go toolchain
# needs/does) and allows our CI builds to be ephemeral.
BINDIR := ./tmp/bin

LINTER := $(BINDIR)/golangci-lint
LINTER_VERSION := 1.38.0

# Linux vs Darwin detection for the machine on which the build is taking place (not to be used for the build target)
DEV_OS := $(shell uname -s | tr A-Z a-z)
$(LINTER):
	mkdir -p $(BINDIR)
	wget "https://github.com/golangci/golangci-lint/releases/download/v$(LINTER_VERSION)/golangci-lint-$(LINTER_VERSION)-$(DEV_OS)-amd64.tar.gz" -O - | tar -xz -C $(BINDIR) --strip-components=1 --exclude=README.md --exclude=LICENSE

# Run go-fmt on all go files.  We list all go files in the repository, run
# gofmt.  gofmt produces output with a list of files that have fmt errors.  If
# we have an empty output, we exit with 0 status, otherwise we exit with nonzero
# status.
.PHONY: fmt-check
fmt-check:
	echo $$(gofmt -l -d $$(find . -type f -name '*.go')) 
	#&& [ -z "$$BADFILES" ] && exit 0

# Run lint checks
.PHONY: lint-check
lint-check: $(LINTER)
	$(LINTER) run
.PHONY: test 
test:
	go test ./...
precommit:
	make lint-check
	make fmt-check
build:
	
	GOOS=darwin GOARCH=arm64 go build -o ./main_darwin ./cmd/main/
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o ./main_linux  ./cmd/main/
	GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -o ./main_windows ./cmd/main/
	
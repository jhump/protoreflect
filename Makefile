.PHONY: ci
ci: deps checkgofmt checkgenerate errcheck golint vet staticcheck ineffassign test

# Lint and codegen tools are pinned in a separate module, so their
# dependencies don't become requirements of this library.
GO_TOOL := go tool -modfile=internal/tools/go.mod

.PHONY: deps
deps:
	go get -d -v -t ./...

.PHONY: updatedeps
updatedeps:
	go get -d -v -t -u -f ./...

.PHONY: install
install:
	go install ./...

.PHONY: checkgofmt
checkgofmt:
	@echo gofmt -s -l .
	@output="$$(gofmt -s -l .)" ; \
	if [ -n "$$output"  ]; then \
	    echo "$$output"; \
		echo "Run gofmt on the above files!"; \
		exit 1; \
	fi

# workaround https://github.com/golang/protobuf/issues/214 until in master
.PHONY: vet
vet:
	go vet ./...

.PHONY: staticcheck
staticcheck:
	$(GO_TOOL) staticcheck ./...

.PHONY: ineffassign
ineffassign:
	$(GO_TOOL) ineffassign ./...

# Intentionally omitted from CI, but target here for ad-hoc reports.
.PHONY: golint
golint:
	$(GO_TOOL) golint -min_confidence 0.9 -set_exit_status ./...

.PHONY: errcheck
errcheck:
	$(GO_TOOL) errcheck ./...

.PHONY: test
test: generate
	go test -cover -race ./...
	./protoprint/testfiles/check-protos.sh > /dev/null

.PHONY: generate
generate:
	go generate ./...
	go generate ./internal/testprotos
	$(GO_TOOL) goimports -w -local github.com/jhump/protoreflect/v2 .

.PHONY: checkgenerate
checkgenerate: generate
	# Make sure generate target doesn't produce a diff
	 test -z "$$(git status --porcelain | tee /dev/stderr)"

.PHONY: ci
ci: deps checkgofmt checkgenerate errcheck golint vet staticcheck ineffassign test

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
	@go install honnef.co/go/tools/cmd/staticcheck@v0.5.1
	staticcheck ./...

.PHONY: ineffassign
ineffassign:
	@go install github.com/gordonklaus/ineffassign@v0.0.0-20200309095847-7953dde2c7bf
	ineffassign .

# Intentionally omitted from CI, but target here for ad-hoc reports.
.PHONY: golint
golint:
	@go install golang.org/x/lint/golint@v0.0.0-20210508222113-6edffad5e616
	golint -min_confidence 0.9 -set_exit_status ./...

.PHONY: errcheck
errcheck:
	@go install github.com/kisielk/errcheck@v1.7.0
	errcheck ./...

.PHONY: test
test: generate
	go test -cover -race ./...
	./protoprint/testfiles/check-protos.sh > /dev/null

.PHONY: generate
generate:
	@go install golang.org/x/tools/cmd/goimports@v0.14.0
	go generate ./...
	go generate ./internal/testdata
	goimports -w -local github.com/jhump/protoreflect/v2 .

.PHONY: checkgenerate
checkgenerate: generate
	# Make sure generate target doesn't produce a diff
	 test -z "$$(git status --porcelain | tee /dev/stderr)"

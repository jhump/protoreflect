# TODO: run golint, errcheck
.PHONY: ci
# TODO: add staticcheck back ASAP; removed temporarily because it
# complains about a lot of APIs deprecated by protobuf 1.4
ci: deps checkgofmt vet ineffassign test test-nounsafe

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
	@echo go vet ./...  --ignore internal/testprotos
	@go vet $$(go list ./... | grep -v 'internal/testprotos')

# goyacc generates assignments where LHS is never used, so we need to run
# staticheck in a way that ignores the errors in that generated code
.PHONY: staticcheck
staticcheck:
	@go install honnef.co/go/tools/cmd/staticcheck@v0.0.1-2020.1.4
	staticcheck ./...

# same remarks as for staticcheck: we ignore errors in generated proto.y.go
.PHONY: ineffassign
ineffassign:
	@go install github.com/gordonklaus/ineffassign@v0.0.0-20200309095847-7953dde2c7bf
	@echo ineffassign . --ignore desc/protoparse/proto.y.go
	@ineffassign -n $$(find . -type d | grep -v 'desc/protoparse')
	@output="$$(ineffassign ./desc/protoparse | grep -v 'protoDollar' || true)" ; \
	if [ -n "$$output"  ]; then \
	    echo "$$output"; \
	    exit 1; \
	fi

# Intentionally omitted from CI, but target here for ad-hoc reports.
.PHONY: golint
golint:
	@go install golang.org/x/lint/golint
	golint -min_confidence 0.9 -set_exit_status ./...

# Intentionally omitted from CI, but target here for ad-hoc reports.
.PHONY: errcheck
errcheck:
	@go install github.com/kisielk/errcheck
	errcheck ./...

.PHONY: test
test:
	go test -cover -race ./...

.PHONY: test-nounsafe
test-nounsafe:
	go test -tags purego -cover -race ./...

.PHONY: generate
generate:
	@go install golang.org/x/tools/cmd/goyacc@v0.0.0-20200717024301-6ddee64345a6
	go generate ./...

.PHONY: testcover
testcover:
	@echo go test -race -covermode=atomic ./...
	@echo "mode: atomic" > coverage.out
	@for dir in $$(go list ./...); do \
		go test -race -coverprofile profile.out -covermode=atomic $$dir ; \
		if [ -f profile.out ]; then \
			tail -n +2 profile.out >> coverage.out && rm profile.out ; \
		fi \
	done


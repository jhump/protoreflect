# TODO: run golint, errcheck
.PHONY: default
default: deps checkgofmt vet predeclared staticcheck ineffassign test

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
	@go get honnef.co/go/tools/cmd/staticcheck
	staticcheck ./...

# same remarks as for staticcheck: we ignore errors in generated proto.y.go
.PHONY: ineffassign
ineffassign:
	@go get github.com/gordonklaus/ineffassign
	@echo ineffassign . --ignore desc/protoparse/proto.y.go
	@ineffassign -n $$(find . -type d | grep -v 'desc/protoparse')
	@output="$$(ineffassign ./desc/protoparse | grep -v 'protoDollar' || true)" ; \
	if [ -n "$$output"  ]; then \
	    echo "$$output"; \
	    exit 1; \
	fi

.PHONY: predeclared
predeclared:
	@go get github.com/nishanths/predeclared
	predeclared .

# Intentionally omitted from CI, but target here for ad-hoc reports.
.PHONY: golint
golint:
	@go get golang.org/x/lint/golint
	golint -min_confidence 0.9 -set_exit_status ./...

# Intentionally omitted from CI, but target here for ad-hoc reports.
.PHONY: errcheck
errcheck:
	@go get github.com/kisielk/errcheck
	errcheck ./...

.PHONY: test
test:
	go test -cover -race ./...

.PHONY: generate
generate:
	@go get golang.org/x/tools/cmd/goyacc
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


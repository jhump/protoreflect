
install:
	go get -v ./...

generate:
	cd internal/testprotos && ./make_protos.sh && cd -

test: 
	./ci.sh

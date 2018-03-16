all: dep fmt lint test bench

dep:
	go get -t -v ./...

fmt:
	@[ $$(gofmt -l . | wc -l) -gt 0 ] && echo "Code differs from gofmt's style" && exit 1 || true

lint:
	# expects golint installed
	# go get github.com/golang/lint/golint
	golint -set_exit_status ./...
	go vet ./...

gocov:
	# expects gocov installed
	#go get github.com/axw/gocov/gocov
	gocov test ./... | gocov report; \
	# gocov test $$(glide novendor) >/tmp/gocovtest.json ; gocov annotate /tmp/gocovtest.json MyFunc

test:
	go test -v ./...

bench:
	@find . -iname '*.go' -exec dirname {} \+ | sort | uniq | while read d ; do pushd $d; go test -bench=. ; popd ; done

build: dep lint test
	go clean -v
	go build -v

install: dep
	go install

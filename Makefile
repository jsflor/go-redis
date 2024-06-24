.PHONY: run
run: build
	./bin/goredis

.PHONY: build
build:
	go build -o bin/goredis .
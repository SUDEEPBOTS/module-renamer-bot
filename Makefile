.PHONY: all build run test clean docker-build

all: build

build:
	go build -o bin/module-renamer-bot cmd/bot/main.go

run:
	go run cmd/bot/main.go

test:
	go test -v ./...

clean:
	rm -rf bin/ /tmp/renamer_work/

docker-build:
	docker build -t sudeepbots/module-renamer-bot:latest .

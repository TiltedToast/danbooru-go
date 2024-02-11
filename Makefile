build:
	go build -o bin/danbooru-go cmd/danbooru-go/main.go

install:
	cd cmd/danbooru-go && go install

run:
	go run cmd/danbooru-go/main.go

test:
	go run cmd/danbooru-go/main.go -t violet_evergarden
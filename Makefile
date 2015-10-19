
test:
	go test ./... -v

test_race:
	go test -v -race ./...

bench:
	go test github.com/mat/heroku-datadog-drain-go -bench .

vet:
	go tool vet *.go

lint:
	golint ./...

style: vet lint

.PHONY: style lint vet bench test_race test


test:
	go test ./... -v

test_race:
	go test -v -race ./...

bench:
	go test github.com/mat/heroku-datadog-drain-go -bench .

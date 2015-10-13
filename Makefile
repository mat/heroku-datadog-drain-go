
test:
	go test ./... -v

bench:
	go test github.com/mat/statslogdrain -bench .

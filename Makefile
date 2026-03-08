run:
	go run .

build:
	go build -o herald .

setup-gmail:
	go run . --setup-gmail

release-dry:
	goreleaser release --snapshot --clean

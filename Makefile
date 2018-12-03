
# Run unit tests
test:
	go test ./...

# Run unit tests with code coverage
test_coverage:
	go test ./... -coverprofile cp.out

PACKAGE_CLIENT = github.com/3scale/3scale-go-client/client

# Run unit tests
test:
	go test $(PACKAGE_CLIENT)

# Run unit tests with code coverage
test_coverage:
	go test $(PACKAGE_CLIENT) -test.coverprofile="coverage.txt"

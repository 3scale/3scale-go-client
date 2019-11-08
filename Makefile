PACKAGE_CLIENT = github.com/3scale/3scale-go-client/threescale

.PHONY: test
test: # Run unit tests
	go test $(PACKAGE_CLIENT)

.PHONY: test_coverage
test_coverage: # Run unit tests with code coverage
	go test $(PACKAGE_CLIENT) -test.coverprofile="coverage.txt"

.PHONY: build swagger

lint:
	golangci-lint run ./...

gen-mocks:
	@mockery --case snake --name=WebSocket

cover:
	@go test $(go list ./... | grep -v /mocks/) -v -covermode=count -coverprofile=coverage.out
	@goveralls -coverprofile=coverage.out -service=travis-ci -repotoken $COVERALLS_TOKEN_SWAGGERWS

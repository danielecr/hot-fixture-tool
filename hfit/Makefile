BINARY_NAME=hfit

.PHONY: build clean install

build:
	go build -o $(BINARY_NAME)

clean:
	go clean
	rm -f $(BINARY_NAME)

install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/

test:
	go test ./...
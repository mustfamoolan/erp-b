.PHONY: build run clean

BINARY_NAME=m3aml-erp

build:
	go build -o bin/$(BINARY_NAME) main.go

run:
	go run main.go

clean:
	rm -rf bin/
	go clean

BINARY_NAME := container-host

.PHONY: build run clean clean_vm

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) *.go

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	go clean

clean_vm:
	@echo "Resetting VM and SSH Keys"
	rm -rf ./images/*
	rm -rf ./ssh_keys/*

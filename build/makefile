APP_NAME=onyrat
BIN_DIR=bin

.PHONY: all client server

all: client server

client:
	garble -seed=random -literals -tiny build -tags=client -o $(BIN_DIR)/$(APP_NAME)-client ./cmd/client

server:
	garble -seed=random -literals -tiny build -tags=server -o $(BIN_DIR)/$(APP_NAME)-server ./cmd/server

clean:
	rm -rf $(BIN_DIR)

.PHONY: ca-gen pk-gen
CA_NAME=ifelser-mitm-ca
EXECUTABLE=main
CERTS_DIR=./certs

all: ca-gen pk-gen build

build:
	@echo "build app"
	go build -o $(EXECUTABLE) ./cmd

ca-gen:
	@echo "generate ca cert"
	mkdir -p $(CERTS_DIR)
	./scripts/gen_ca.sh $(CERTS_DIR)/$(CA_NAME)

pk-gen:
	@echo "generate private key"
	mkdir -p "$(CERTS_DIR)"
	openssl genrsa -out "$(CERTS_DIR)/cert.key" 2048

clean:
	rm -rf $(EXECUTABLE) $(CERTS_DIR)

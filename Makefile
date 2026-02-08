EXECUTABLE=mitm-proxy

CA_NAME=ifelser-mitm-ca
CERTS_DIR=./certs

.PHONY: ca-gen pk-gen

all: ca-gen pk-gen build

build:
	@echo "build app"
	go build -o $(EXECUTABLE) ./cmd/main.go

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

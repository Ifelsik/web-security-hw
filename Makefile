CA_NAME=ca
EXECUTABLE=main

all: ca-gen build

build:
	echo "build app"
	go build -o $(EXECUTABLE) ./cmd

ca-gen:
	echo "generate ca cert"
	./scripts/gen_ca.sh $(CA_NAME)

clean:
	rm -rf $(EXECUTABLE) $(CA_NAME).*

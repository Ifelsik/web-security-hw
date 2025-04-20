FROM golang:1.24

WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/proxy
CMD ["./main"]

EXPOSE 8080 8000

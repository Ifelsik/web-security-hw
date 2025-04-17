FROM golang:1.24

WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd
CMD ["./main"]

EXPOSE 8080

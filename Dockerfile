FROM golang:1.25 AS build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o mitm-proxy ./cmd/main.go

FROM alpine:3.22

WORKDIR /app

COPY --from=build /build/mitm-proxy ./cmd/

COPY --from=build /build/certs ./certs/

ENTRYPOINT ["./cmd/mitm-proxy"]

EXPOSE 8080 8000

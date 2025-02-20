FROM golang:1.23-alpine

WORKDIR /app

RUN apk --no-cache add bash gcc musl-dev

# dependencies
COPY ["app/go.mod", "app/go.sum", "./"]
RUN go mod download

# build
COPY ["app/", "./"]
RUN CGO_ENABLED=0 GOOS=linux go build -o tasks main.go

CMD ["./tasks"]

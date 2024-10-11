# File: service-b/Dockerfile

FROM golang:1.22-alpine

WORKDIR /app

# Install Git and certificates
RUN apk update && apk add --no-cache git ca-certificates

# Copy go.mod and go.sum
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the application
RUN go build -o service-b .

EXPOSE 8081

CMD ["./service-b"]

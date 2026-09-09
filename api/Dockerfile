FROM golang:1.21-alpine

WORKDIR /app

# Copy go.mod and go.sum first (for dependency caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the binary
RUN go build -o api main.go

EXPOSE 8080
CMD ["./api"]
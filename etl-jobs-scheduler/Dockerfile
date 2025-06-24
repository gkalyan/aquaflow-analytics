FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the scheduler
RUN CGO_ENABLED=0 GOOS=linux go build -o scheduler ./cmd/scheduler

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary
COPY --from=builder /app/scheduler .

# Ensure the binary is executable
RUN chmod +x ./scheduler

CMD ["./scheduler"]
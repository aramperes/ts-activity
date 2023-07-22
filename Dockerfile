FROM golang:latest as builder

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/ts-activity

# Runner image
FROM alpine:latest

RUN adduser --disabled-password tsactivity
RUN apk --no-cache add dumb-init
WORKDIR /home/tsactivity

COPY --from=builder /app/ts-activity /home/tsactivity/ts-activity
RUN chmod +x /home/tsactivity/ts-activity

# Run
USER tsactivity
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/home/tsactivity/ts-activity"]

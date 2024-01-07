# Build stage
FROM golang:1.21-alpine AS build-env

# Set up necessary Go environment variables (optional, based on your app's needs)
# ENV GO111MODULE=on \
#     CGO_ENABLED=0 \
#     GOOS=linux \
#     GOARCH=amd64

# Set the working directory inside the container
WORKDIR /app

# Copy the source from the current directory to the working Directory inside the container
COPY . .

# Build the Go app
RUN go mod tidy && go mod vendor && go build -o fern .

# Final stage: Run stage
FROM alpine

# Copy the pre-built binary file from the previous stage
COPY --from=build-env /app/fern /app/fern

# Set the working directory in the container
WORKDIR /app

# Command to run the binary
CMD ["/app/fern"]


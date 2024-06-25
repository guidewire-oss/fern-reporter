# Build stage
FROM --platform=${BUILDPLATFORM} golang:1.21-alpine AS build-env

# Set up arguments for multi-architecture support
ARG TARGETOS
ARG TARGETARCH

# Set up the environment
ENV GO111MODULE=on \
    CGO_ENABLED=0

RUN apk --no-cache add ca-certificates \
  && update-ca-certificates

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod tidy && go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .
COPY config/config.yaml ./
RUN mkdir ./migrations
COPY pkg/db/migrations ./migrations/

# Build the Go app
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o fern .

# Final stage
FROM --platform=${TARGETPLATFORM} alpine
WORKDIR /app

# Copy the binary from the build-env
COPY --from=build-env /app/fern /app/
COPY --from=build-env /app/config.yaml /app/
RUN mkdir /app/migrations
COPY --from=build-env /app/migrations/* /app/migrations/

# Command to run
ENTRYPOINT ["/app/fern"]

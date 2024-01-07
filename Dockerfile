# Build stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.21 as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV GO111MODULE=on

# Set the working directory inside the container
WORKDIR /app

# Copy the source from the current directory to the working Directory inside the container
COPY . .
RUN go mod download

# Build the Go app
# RUN go mod tidy && go build -o fern .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o fern .

# Final stage: Run stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot


# Set the working directory in the container
WORKDIR /app

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/fern /app/fern

USER nonroot:nonroot

# Command to run the binary
CMD ["/app/fern"]


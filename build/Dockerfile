# syntax=docker/dockerfile:1.4

############################
# STEP 1 build executable binary
############################
# Download the base image for building Go apps
FROM golang:1.22 as builder
# Set the working directory
WORKDIR /app
# Copy the code into the container
COPY . .
# Build the Go binary
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG GO_BINARY
ENV GO_BINARY ${GO_BINARY:-./cmd/printersmtpserver}
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -mod vendor -o ./bin/app $GO_BINARY

############################
# STEP 2 build a small image
############################
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/bin/app /app

# Create the attachments directory and set the environment variable
ENV SMTP_OUTPUTPATH=/attachments
ENV SMTP_PORT=25000

# Expose port 25000
EXPOSE 25000

ENTRYPOINT ["/app"]
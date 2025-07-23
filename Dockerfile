# Build the application from source
FROM golang:1.24-alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /loadtest

# Run the tests in the container
# FROM build-stage AS run-test-stage
# RUN go test -v ./...

# Deploy the application binary into a lean image
FROM alpine:3.21 AS build-release-stage

WORKDIR /app

COPY --from=build-stage /loadtest /loadtest

EXPOSE 8080

# USER nonroot:nonroot

CMD ["/loadtest"]
# Start from the latest golang base image
FROM golang:1-alpine AS builder

LABEL maintainer="cgil"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/notes-server ./notes-server
COPY pkg ./pkg
COPY gen ./gen

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o notes-server ./notes-server

######## Start a new stage  #######
FROM scratch
USER 1221:1221
WORKDIR /goapp

COPY --from=builder /app/notes-server .

ENV PORT="${PORT}"
ENV DB_DRIVER="${DB_DRIVER}"
ENV DB_HOST="${DB_HOST}"
ENV DB_PORT="${DB_PORT}"
ENV DB_NAME="${DB_NAME}"
ENV DB_USER="${DB_USER}"
ENV DB_PASSWORD="${DB_PASSWORD}"
ENV DB_SSL_MODE="${DB_SSL_MODE}"

EXPOSE 8080

HEALTHCHECK --start-period=5s --interval=30s --timeout=3s \
    CMD curl --fail http://localhost:8080/health || exit 1

CMD ["./notes-server"]

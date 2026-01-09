# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

ARG SERVICE
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/${SERVICE} ./cmd/${SERVICE}

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

ARG SERVICE
COPY --from=build /out/${SERVICE} /app/${SERVICE}
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

ENV HTTP_ADDR=:8080
ENV GRPC_ADDR=:50051
ENV SERVICE=""

EXPOSE 8080 50051

ENTRYPOINT ["/app/entrypoint.sh"]

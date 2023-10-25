# syntax=docker/dockerfile:1

FROM golang:1.21.3-alpine3.18

ARG PORT
ENV PORT=$PORT

WORKDIR /app
COPY . ./
WORKDIR /app/api_ws
RUN CGO_ENABLED=0 GOOS=linux go build -o /api_ws_bin
CMD ["/api_ws_bin"]

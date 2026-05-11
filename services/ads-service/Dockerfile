FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/ads-service ./cmd

FROM alpine:3.20
RUN apk add --no-cache wget
WORKDIR /app
COPY --from=build /out/ads-service ./ads-service
ENV HTTP_ADDR=:8081
EXPOSE 8081
USER nobody
ENTRYPOINT ["/app/ads-service"]

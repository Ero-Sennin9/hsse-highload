FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/billing-service ./cmd

FROM alpine:3.20
RUN apk add --no-cache wget
WORKDIR /app
COPY --from=build /out/billing-service ./billing-service
ENV HTTP_ADDR=:8082
EXPOSE 8082
USER nobody
ENTRYPOINT ["/app/billing-service"]

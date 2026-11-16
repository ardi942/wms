FROM golang:1.24-alpine AS build
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM alpine:3.21
RUN addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app \
 && apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build /out/ /app/
COPY migrations /app/migrations
RUN mkdir -p /app/uploads && chown 10001:10001 /app/uploads
USER 10001:10001
EXPOSE 8080
CMD ["/app/api"]

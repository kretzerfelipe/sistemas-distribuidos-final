FROM golang:1.27-alpine AS build
WORKDIR /src/services/notification
COPY services/notification/go.mod services/notification/go.sum ./
RUN go mod download
COPY services/notification/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.20
RUN adduser -D -u 10001 app
COPY --from=build /out/api /usr/local/bin/api
COPY contracts /app/contracts
ENV CONTRACTS_DIR=/app/contracts
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["/usr/local/bin/api"]

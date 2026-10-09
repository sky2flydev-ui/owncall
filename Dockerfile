FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /owncall ./cmd/server

FROM alpine:3.20
RUN addgroup -S owncall && adduser -S -G owncall owncall
WORKDIR /app
COPY --from=build /owncall /usr/local/bin/owncall
COPY web ./web
USER owncall
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/owncall"]

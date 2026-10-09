FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /owncall ./cmd/server
FROM alpine:3.20
WORKDIR /app
COPY --from=build /owncall /usr/local/bin/owncall
COPY web ./web
EXPOSE 8080
CMD ["owncall"]

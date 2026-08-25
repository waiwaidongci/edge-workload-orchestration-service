FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /edge-task-orchestrator ./cmd/edge-task-orchestrator
FROM alpine:3.20
RUN adduser -D -u 10001 edge
COPY --from=build /edge-task-orchestrator /usr/local/bin/edge-task-orchestrator
USER edge
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/edge-task-orchestrator"]


# gfd image: DragonsNShit MUD + real Go backend (server-go). One image, two binaries; the cluster pod
# (EMILY/gitops/specs/gfd-core.pod) runs them as sibling containers. WORKDIR /app so the relative
# data/ and var/ paths resolve; var/ is the PVC mount.
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY . .
RUN GOWORK=off CGO_ENABLED=0 go build -o /out/mud ./apps2/mud \
 && GOWORK=off CGO_ENABLED=0 go build -o /out/server-go ./apps2/server-go
FROM debian:12-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/mud /out/server-go /app/
COPY data /app/data
COPY ops/genesis /app/ops/genesis

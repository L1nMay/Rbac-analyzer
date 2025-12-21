# --- build stage ---
FROM golang:1.22 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/rbac-server ./cmd/rbac-server

# --- runtime stage ---
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/rbac-server /rbac-server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/rbac-server"]

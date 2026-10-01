# syntax=docker/dockerfile:1
# One Dockerfile builds every service: docker build --build-arg SERVICE=order-service .

FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY pkg ./pkg
ARG SERVICE
COPY ${SERVICE} ./${SERVICE}
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./${SERVICE}

# distroless: no shell, no package manager, runs as an unprivileged user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=5 CMD ["/app", "healthcheck"]
ENTRYPOINT ["/app"]

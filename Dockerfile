FROM golang:1.26 AS build

ARG VERSION=dev
ARG TARGETARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -o /out/agent ./cmd/agent

FROM gcr.io/distroless/static-debian12:latest

ARG VERSION=dev

LABEL org.opencontainers.image.source="https://github.com/jrzayev/kubenpu" \
      org.opencontainers.image.description="Per-pod visibility into NPU and accelerator usage on Kubernetes" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}"

ENV KUBENPU_APP_VERSION=${VERSION}

COPY --from=build /out/agent /agent

ENTRYPOINT ["/agent"]

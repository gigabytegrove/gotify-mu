ARG BUILDKIT_SBOM_SCAN_CONTEXT=true
# Suppress warning about invalid variable expansion
ARG GO_VERSION=PLEASE_PROVIDE_GO_VERSION
ARG DEBIAN=sid-slim

# Hack to normalize platform to match the chosen build image
ARG __TARGETPLATFORM_DASHES=${TARGETPLATFORM/\//-}
ARG __TARGETPLATFORM_GO_NOTATION=${__TARGETPLATFORM_DASHES/arm\/v7/arm-7}

# --- JS Builder ---

FROM --platform=${BUILDPLATFORM} node:24 AS js-builder

ARG BUILD_JS=0

COPY ./Makefile /src/monita/Makefile
COPY ./ui /src/monita/ui

RUN if [ "$BUILD_JS" = "1" ]; then \
    (cd /src/monita/ui && yarn install) && \
    (cd /src/monita && make build-js) \
    else \
    mkdir -p /src/monita/ui/build; \
    fi

# --- Go Builder ---

FROM --platform=${BUILDPLATFORM} gotify/build:${GO_VERSION}-${__TARGETPLATFORM_GO_NOTATION} AS builder

ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG BUILD_JS=0
ARG RUN_TESTS=0
ARG MONITA_VERSION=
ARG MONITA_COMMIT=
ARG MONITA_BUILD_DATE=
ARG GOTIFY_MU_VERSION=
ARG GOTIFY_MU_COMMIT=
ARG GOTIFY_MU_BUILD_DATE=
ARG LD_FLAGS=""
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -yq --no-install-recommends \
    ca-certificates \
    git

COPY . /src/monita
COPY --from=js-builder /src/monita/ui/build /ui-build

RUN if [ "$BUILD_JS" = "1" ]; then \
    cp -r --update /ui-build /src/monita/ui/build; \
    fi

RUN cd /src/monita && \
    if [ "$RUN_TESTS" = "1" ] && [ "$BUILDPLATFORM" = "$TARGETPLATFORM" ]; then \
    go test -v ./...; \
    fi && \
    EFFECTIVE_LD_FLAGS="$LD_FLAGS" && \
    if [ -z "$EFFECTIVE_LD_FLAGS" ]; then \
    EFFECTIVE_VERSION="${MONITA_VERSION:-${GOTIFY_MU_VERSION:-master-local}}"; \
    EFFECTIVE_COMMIT="${MONITA_COMMIT:-${GOTIFY_MU_COMMIT:-local}}"; \
    EFFECTIVE_BUILD_DATE="${MONITA_BUILD_DATE:-${GOTIFY_MU_BUILD_DATE:-local}}"; \
    EFFECTIVE_LD_FLAGS="-w -s -X main.Version=${EFFECTIVE_VERSION} -X main.Commit=${EFFECTIVE_COMMIT} -X main.BuildDate=${EFFECTIVE_BUILD_DATE} -X main.Mode=prod"; \
    fi && \
    LD_FLAGS="$EFFECTIVE_LD_FLAGS" make OUTPUT=/target/app/monita-app _build_within_docker && \
    ln -s monita-app /target/app/gotify-app

# --- Runtime ---

FROM debian:${DEBIAN}

ARG MONITA_VERSION=master-local
ARG MONITA_COMMIT=local

LABEL org.opencontainers.image.documentation=https://github.com/gigabytegrove/monita
LABEL org.opencontainers.image.source=https://github.com/gigabytegrove/monita
LABEL org.opencontainers.image.title="Monita"
LABEL org.opencontainers.image.description="Monita self-hosted notifications, messaging, and automation platform"
LABEL org.opencontainers.image.version=${MONITA_VERSION}
LABEL org.opencontainers.image.revision=${MONITA_COMMIT}
LABEL org.opencontainers.image.licenses="MIT"

ARG GOTIFY_SERVER_EXPOSE=80
ARG MONITA_SERVER_EXPOSE=$GOTIFY_SERVER_EXPOSE
ENV MONITA_SERVER_PORT=$MONITA_SERVER_EXPOSE

WORKDIR /app

RUN export DEBIAN_FRONTEND=noninteractive && apt-get update && apt-get install -yq --no-install-recommends \
    tzdata \
    curl \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s CMD curl --fail http://localhost:$MONITA_SERVER_PORT/health || exit 1
EXPOSE $MONITA_SERVER_EXPOSE

COPY --from=builder /target /

COPY --chmod=0755 docker/monita-entrypoint.sh /usr/local/bin/monita-entrypoint

ENTRYPOINT ["/usr/local/bin/monita-entrypoint"]
CMD ["serve"]

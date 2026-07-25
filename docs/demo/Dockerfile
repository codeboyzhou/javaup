# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.5

FROM golang:${GO_VERSION}-bookworm AS jup-builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/jup ./cmd/jup

FROM ghcr.io/charmbracelet/vhs:v0.11.0

USER root

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        fontconfig \
        maven \
        unzip \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /usr/local/share/fonts/LiterationMono \
    && curl --fail --location --retry 3 \
        "https://github.com/ryanoasis/nerd-fonts/releases/latest/download/LiberationMono.zip" \
        --output /tmp/LiberationMono.zip \
    && unzip -q /tmp/LiberationMono.zip -d /usr/local/share/fonts/LiterationMono \
    && rm /tmp/LiberationMono.zip \
    && fc-cache --force

RUN mkdir -p /opt/jdks/java8 /opt/jdks/java17 \
    && curl --fail --location --retry 3 \
        "https://api.adoptium.net/v3/binary/latest/8/ga/linux/x64/jdk/hotspot/normal/eclipse" \
        | tar --extract --gzip --strip-components=1 --directory=/opt/jdks/java8 \
    && curl --fail --location --retry 3 \
        "https://api.adoptium.net/v3/binary/latest/17/ga/linux/x64/jdk/hotspot/normal/eclipse" \
        | tar --extract --gzip --strip-components=1 --directory=/opt/jdks/java17

# Debian's Maven emits ANSI styling even when jup captures `mvn --version`.
# Keep Maven's real output while removing those control bytes so jup can parse it.
RUN <<'EOF'
cat >/usr/local/bin/mvn <<'SCRIPT'
#!/usr/bin/env bash
set -o pipefail

if [[ "${1:-}" == "--version" || "${1:-}" == "-version" || "${1:-}" == "-v" ]]; then
  /usr/bin/mvn "$@" | sed --regexp-extended 's/\x1B\[[0-9;]*[mK]//g'
  exit $?
fi

exec /usr/bin/mvn "$@"
SCRIPT
chmod +x /usr/local/bin/mvn
EOF

COPY --from=jup-builder /out/jup /usr/local/bin/jup

ENV JAVA_HOME=/opt/jdks/java17
ENV JAVA17_HOME=/opt/jdks/java17
ENV JAVA8_HOME=/opt/jdks/java8
ENV PATH=/opt/jdks/java17/bin:/usr/local/bin:/usr/local/sbin:/usr/sbin:/usr/bin:/sbin:/bin

# Cache all Maven plugins needed by the tape before recording starts.
COPY docs/demo/java8-app /tmp/demo/java8-app
COPY docs/demo/java17-app /tmp/demo/java17-app
RUN mvn --quiet --file /tmp/demo/java8-app/pom.xml clean test \
    && mvn --quiet --file /tmp/demo/java17-app/pom.xml clean test \
    && rm -rf /tmp/demo

WORKDIR /jup

ENTRYPOINT ["vhs"]

FROM ubuntu:26.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git sudo jq tar gzip build-essential gnupg \
    && rm -rf /var/lib/apt/lists/*
# Go toolchain (arm64 native; can cross-compile to linux/amd64 via GOARCH)
RUN curl -fsSL https://go.dev/dl/go1.26.2.linux-arm64.tar.gz | tar -C /usr/local -xz
ENV PATH=/usr/local/go/bin:/home/runner/.local/bin:$PATH
RUN useradd -m runner && echo 'runner ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/runner
WORKDIR /home/runner
RUN RUNNER_VERSION=2.334.0 && \
    curl -fsSL -o actions-runner.tar.gz "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-arm64-${RUNNER_VERSION}.tar.gz" && \
    tar xzf actions-runner.tar.gz && rm actions-runner.tar.gz && \
    ./bin/installdependencies.sh && \
    rm -rf /var/lib/apt/lists/* && \
    chown -R runner:runner /home/runner
COPY --chown=runner:runner entrypoint.sh /home/runner/entrypoint.sh
USER runner
ENTRYPOINT ["/home/runner/entrypoint.sh"]

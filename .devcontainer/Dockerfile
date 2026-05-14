FROM mcr.microsoft.com/devcontainers/go:1

RUN apt-get update \
  && apt-get install -y --no-install-recommends docker.io \
  && rm -rf /var/lib/apt/lists/*

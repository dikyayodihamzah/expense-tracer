FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bot ./cmd/bot

# Runtime needs Node only for the claude CLI (VISION_PROVIDER=claude-cli).
FROM node:22-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN npm install -g @anthropic-ai/claude-code@2.1.302 && npm cache clean --force
COPY --from=build /bot /usr/local/bin/bot
USER node
WORKDIR /home/node
ENTRYPOINT ["bot"]

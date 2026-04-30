**Weavedeck** is a personal, local-first tool that helps you collect and revisit curated lists from across the internet — like *“100 must-read books”* or *“Top 10 sci-fi movies”*.

Instead of bookmarking and forgetting, Weavedeck quietly builds your own structured library of things you actually wanted to explore.

## Features
- Aggregate curated lists from multiple sources
- Store everything locally
- Built-in search for quick re-discovery
- Categories and tags for organization
- Fast static site powered by Hugo

## Installation (Docker)

The easiest way to run Weavedeck is via Docker.

### 1. Create a `docker-compose.yml`

```yaml
---
services:
  weavedeck:
    image: ghcr.io/evolvedevlab/weavedeck:latest
    container_name: weavedeck
    restart: unless-stopped
    ports:
      - "3000:3000" # change to "4000:3000" if needed
    environment:
      - PUID=1000
      - PGID=1000
      - REBUILD_INTERVAL=10 # in seconds
    volumes:
      - site_data:/app/site
      - ./hugo.toml:/app/site/hugo.toml

volumes:
  site_data:
```

### 2. Add `hugo.toml`

```sh
wget https://github.com/evolvedevlab/weavedeck/raw/refs/heads/main/site/hugo.toml
```

You can tweak the site by editing `hugo.toml`.

### 3. Run the container

```bash
docker compose up -d
```

Now open:
👉 http://localhost:3000

## Build from Source

### Requirements
- Go **1.25.0**

### Steps

```bash
git clone https://github.com/evolvedevlab/weavedeck.git
cd weavedeck
make simple
```

## Running Tests

```bash
make test
```

+++
title = "Docker"
+++

LDAPEnforcer can be run in a Docker container, which is often simpler to deploy in production environments.
The Docker container is published to
[`ghcr.io/mrled/ldapenforcer`](https://github.com/mrled/ldapenforcer/pkgs/container/ldapenforcer),
with a tag for each published version and a `latest` tag for the latest version.

## Quick Start

1. Build the Docker image:
   ```bash
   docker build -t ldapenforcer .
   ```

2. Create a configuration file `config.toml` with your LDAP settings.

3. Run the container:
   ```bash
   docker run -v $(pwd)/config.toml:/etc/ldapenforcer/config.toml \
     ghcr.io/mrled/ldapenforcer:latest \
     ldapenforcer sync --config /etc/ldapenforcer/config.toml
   ```

## Running with Docker Compose

```yaml
version: '3.8'

services:
  ldapenforcer:
    image: ghcr.io/mrled/ldapenforcer:latest
    volumes:
      - ./config.toml:/etc/ldapenforcer/config.toml:ro
    command: ["sync", "--config", "/etc/ldapenforcer/config.toml", "--poll"]
    restart: unless-stopped
    # Optional environment variables for configuration - override config file values
    environment:
      - LDAPENFORCER_LOG_LEVEL=INFO
      # - LDAPENFORCER_URI=ldap://example.com:389
      # - LDAPENFORCER_BIND_DN=cn=admin,dc=example,dc=com
      # - LDAPENFORCER_PASSWORD=changeme
```

1. Configure your LDAP settings in a `config.toml` file.

2. Start the service:
   ```bash
   docker-compose up -d
   ```

3. View logs:
   ```bash
   docker-compose logs -f
   ```

## Configuration

The container expects your configuration file to be mounted at `/etc/ldapenforcer/config.toml`.

You can also use environment variables to configure LDAPEnforcer. Environment variables take precedence over values in the config file.

Example:
```bash
docker run -v $(pwd)/config.toml:/etc/ldapenforcer/config.toml \
  -e LDAPENFORCER_LOG_LEVEL=INFO \
  ldapenforcer sync --config /etc/ldapenforcer/config.toml --poll
```

`ldapenforcer` in the Docker image runs as UID 1000.

# Docker Usage Instructions

LDAPEnforcer can be run in a Docker container, which is often simpler to deploy in production environments.

## Quick Start

1. Build the Docker image:
   ```bash
   docker build -t ldapenforcer .
   ```

2. Create a configuration file `config.toml` with your LDAP settings.

3. Run the container:
   ```bash
   docker run -v $(pwd)/config.toml:/etc/ldapenforcer/config.toml \
     ldapenforcer sync --config /etc/ldapenforcer/config.toml
   ```

## Running with Docker Compose

A `docker-compose.yml` file is provided for convenience.

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
  ldapenforcer sync --config /etc/ldapenforcer/config.toml --poll 10
```

## Using the Polling Feature

To have LDAPEnforcer continuously monitor your LDAP directory and configuration files for changes:

```bash
docker run -v $(pwd)/config.toml:/etc/ldapenforcer/config.toml \
  ldapenforcer sync --config /etc/ldapenforcer/config.toml --poll 10
```

This will:
1. Perform an initial sync
2. Check every 10 seconds if any config files have changed
3. When changes are detected, reload the config and perform another sync

## Security Considerations

The Docker image:
- Runs as a non-root user (UID 10000)
- Contains only the compiled binary and necessary runtime dependencies
- Does not include any development tools or unnecessary packages
- Uses multi-stage builds to minimize attack surface

## Image Size Optimization

The image is optimized for size by:
- Using Alpine Linux as the base (very small footprint)
- Using multi-stage builds to exclude build dependencies
- Stripping debug information from the binary
- Including only necessary runtime files
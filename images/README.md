# Specialized images for initia

## `private`

Use a GitHub PAT (Personal Access Token) to access private git repos releases and git trees.
To be used for releases until repos become public.

Build like:

``` bash
docker build \
    -t initiad \
    --build-arg LIBMOVEVM_VERSION=v1.0.0 \
    --build-arg GITHUB_ACCESS_TOKEN=$PAT \
    -f images/private/Dockerfile \
    .
```

## `node`

Image with custom entrypoint to manage node initialization and facilitate node ops.

Some things it does:

- Init initia base config
- Download genesis from url if not already done
- Set moniker to machine/container hostname for better tracing

The image is supposed to be used with [environment variables](https://docs.cosmos.network/v0.45/core/cli.html#environment-variables) CLI overrides.

Build like:

``` bash
docker build \
    -t node \
    --build-arg BASE_IMAGE=ghcr.io/initia-labs/initiad:v1.0.0 \
    -f images/node/Dockerfile \
    .
```

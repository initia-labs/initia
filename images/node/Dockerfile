ARG BASE_IMAGE=ghcr.io/initia-labs/initiad:latest

FROM "${BASE_IMAGE}"

# defaults for initiad env vars
# see https://docs.cosmos.network/v0.45/core/cli.html#environment-variables
# for a full list of cli flags, run `initiad <command> --help`
ENV INITIAD_HOME="/initia/.initia"
ENV INITIAD_PRUNING=nothing
ENV INITIAD_MINIMUM_GAS_PRICES=0uinit
ENV INITIAD_HALT_HEIGHT=0

COPY images/node/entrypoint.sh ./

ENTRYPOINT ["/initia/entrypoint.sh"]

CMD ["initiad", "start"]

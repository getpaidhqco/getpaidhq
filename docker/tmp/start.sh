set -eu -o pipefail

export TEMPORAL_BROADCAST_ADDRESS=$(hostname -i)

export PROMETHEUS_ENDPOINT=$TEMPORAL_BROADCAST_ADDRESS:8233

sh ./entrypoint.sh
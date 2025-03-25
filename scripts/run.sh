#!/bin/bash

command="$1"

if [ -z "$command" ]; then
  command="start"
fi

ProjectRoot="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export AMBULANCE_API_ENVIRONMENT="Development"
export AMBULANCE_API_PORT="8080"

case "$command" in
  start)
    go run "${ProjectRoot}/cmd/ambulance-api-service"
    ;;
  openapi)
    docker run --rm -ti -v "${ProjectRoot}:/local" openapitools/openapi-generator-cli generate -c /local/scripts/generator-cfg.yaml
    ;;
  *)
    echo "Unknown command: $command"
    exit 1
    ;;
esac
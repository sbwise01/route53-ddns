#!/usr/bin/env bash

set -e

if [ "$R53_DOMAIN" == "" ]; then
    echo "ERROR R53_DOMAIN is mandatory."
    echo "Use --env with docker run (or env in the Kubernetes manifest) to pass it."
    exit 1
fi

ARGS=(--domain="$R53_DOMAIN")

if [ "$R53_HOST" != "" ]; then
    ARGS+=(--host="$R53_HOST")
fi

if [ "$R53_HOSTED_ZONE_ID" != "" ]; then
    ARGS+=(--zone-id="$R53_HOSTED_ZONE_ID")
fi

if [ "$R53_TTL" != "" ]; then
    ARGS+=(--ttl="$R53_TTL")
fi

if [ "$R53_POLL_INTERVAL" != "" ]; then
    ARGS+=(--poll-interval="$R53_POLL_INTERVAL")
fi

exec /app/r53ddns "${ARGS[@]}"

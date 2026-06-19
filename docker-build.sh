#!/usr/bin/env bash

if [ $# -ne 1 ]; then
    echo "Exactly one argument required"
    echo "bash docker-build.sh VERSION"
    echo "  e.g. bash docker-build.sh 1.0.0"
    exit 1
fi

docker buildx create --use --name mybuild
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 --build-arg VERSION=$1 -t sbwise/route53-ddns:$1 --push --pull .
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 --build-arg VERSION=$1 -t sbwise/route53-ddns:latest --push --pull .

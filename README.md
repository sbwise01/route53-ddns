# Route53 DDNS client

A lightweight Dynamic DNS client for AWS Route53. It monitors the public (WAN)
IP of the network it runs on and, whenever that IP differs from the value of an
`A` record in a Route53 hosted zone, it updates the record to match.

This is a Route53 port of the
[namecheap-ddns-docker](https://github.com/navilg/namecheap-ddns-docker) project.

## Why use it?

* Easy to set up
* Lightweight (single static Go binary on Alpine)
* No cronjob configuration required (built-in poll loop)
* Logs to stdout for visibility
* Works with EKS IRSA so no static AWS credentials are needed in-cluster

## How it works

On startup, and then once per poll interval (15 minutes by default), the client:

1. Fetches the current public IP (via `api.ipify.org`, falling back to
   `ipinfo.io`).
2. Reads the current `A` record value from Route53.
3. UPSERTs the record if the values differ.

Because Route53 is the source of truth, there is no local cache to get out of
sync — the record value is always compared directly against the live zone.

## Configuration

The application is configured through command-line flags, or the equivalent
environment variables consumed by the container entrypoint:

| Flag         | Env var               | Required | Default     | Description                                                        |
| ------------ | --------------------- | -------- | ----------- | ------------------------------------------------------------------ |
| `--domain`   | `R53_DOMAIN`          | yes      | —           | Hosted zone domain name, e.g. `example.com`                        |
| `--host`     | `R53_HOST`            | no       | apex/root   | Subdomain/hostname, e.g. `home`. Use `@` or omit for the apex.     |
| `--zone-id`  | `R53_HOSTED_ZONE_ID`  | no       | looked up   | Hosted zone ID. When omitted it is resolved from `--domain`.       |
| `--ttl`      | `R53_TTL`             | no       | `300`       | TTL in seconds for the managed record.                             |
| `--poll-interval` | `R53_POLL_INTERVAL` | no   | `15m`       | How often to reconcile, as a Go duration, e.g. `15m`, `5m`, `30s`. |

AWS credentials and region are resolved via the standard AWS credential chain.
A region must be available (e.g. `AWS_REGION=us-east-1`).

## AWS permissions

The identity used by the client needs the following Route53 permissions:

* `route53:ListHostedZonesByName` (only required when the zone ID is looked up)
* `route53:ListResourceRecordSets`
* `route53:ChangeResourceRecordSets`

## Running with Docker

```
docker run --name home.example.com -d --restart unless-stopped \
  -e R53_HOST='home' \
  -e R53_DOMAIN='example.com' \
  -e AWS_REGION='us-east-1' \
  -e AWS_ACCESS_KEY_ID='...' \
  -e AWS_SECRET_ACCESS_KEY='...' \
  sbwise/route53-ddns
```

Or with `docker compose` (see `docker-compose.yml`):

```
docker compose up -d
docker compose logs -f
```

## Running on Kubernetes (EKS) with IRSA

`route53-ddns.yaml` contains a `Namespace`, an IRSA-annotated `ServiceAccount`,
and a `Deployment`. No AWS secret is needed — the pod assumes the IAM role via
its service account.

1. Create an IAM role with the Route53 permissions listed above and a trust
   policy that allows the `route53-ddns` service account in the `route53-ddns`
   namespace.
2. Set the role ARN in the `eks.amazonaws.com/role-arn` annotation on the
   `ServiceAccount`.
3. Adjust the `R53_HOST` / `R53_DOMAIN` env values.
4. Apply it:

```
kubectl apply -f route53-ddns.yaml
kubectl -n route53-ddns logs -f deploy/route53-ddns
```

## Build your own image

The release version lives in `VERSION` (semver `X.Y.Z`). CI reads it on merge to
`main`, tags the repo, and pushes `sbwise/route53-ddns:<VERSION>`.

```
docker build --build-arg VERSION="$(cat VERSION)" -t sbwise/route53-ddns .
```

Or build/push multi-arch images:

```
bash docker-build.sh "$(cat VERSION)"
```

## CI/CD

PRs that touch Go sources, the Dockerfile, or `VERSION` run a Docker test build
and bump `VERSION` from the PR title (Conventional Commits: `feat` → minor,
everything else → patch, `!` / `BREAKING CHANGE` → major). The bot commits the
bump with `[skip ci]`.

On merge to `main`, release tags `VERSION` and pushes the image via OIDC →
`dockerhub-role` → SSM (no GitHub Docker Hub secret).

Before the first PR after adding CI, seed the baseline tag on `main`:

```
git tag 2.0.0 origin/main && git push origin 2.0.0
```

Flux in `wise-k8s` tracks `sbwise/route53-ddns` with semver `>=2.0.0` and bumps
the cluster overlay automatically.

## Build locally

```
go build -o r53ddns .
./r53ddns --domain example.com --host home
```

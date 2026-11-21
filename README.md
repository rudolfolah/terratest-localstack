# terratest-localstack
Run the same Terraform/S3 integration tests against LocalStack or AWS Emulator.

Based on the following documentation and code:
- ["Terraform (IaC) testing: Localstack + Terratest.", Corentin Le Devedec](https://medium.com/@ledevedeccorentin/terraform-iac-testing-localstack-terratest-9946dafe98b6)
  - [ldcorentin/aws-terratest-localstack](https://github.com/ldcorentin/aws-terratest-localstack)
- [edsoncelio/terratest-localstack](https://github.com/edsoncelio/terratest-localstack)
- [icarrera/terratest-localstack-example](https://github.com/icarrera/terratest-localstack-example)

## Integration tests

The [backend manifest](./infra/test-backends.json) supplies endpoint and emulator
defaults, plus LocalStack's dummy credentials. The [runner](./scripts/test-infra.sh)
selects the backend with `TEST_BACKEND`; unset or empty selects `localstack`.
Both tests use the same [Terraform provider](./infra/provider.tf) and use Go's AWS
SDK for their S3 assertions, one through Terratest and one directly.

The infrastructure is based on the examples from terratest:
- [terraform-aws-s3-example](https://github.com/gruntwork-io/terratest/tree/master/examples/terraform-aws-s3-example)
- [terraform-aws-rds-example](https://github.com/gruntwork-io/terratest/tree/master/examples/terraform-aws-rds-example)

### LocalStack (default)

Install [Docker Compose](https://docs.docker.com/compose/install/),
[Terraform](https://developer.hashicorp.com/terraform/install) and
[Go](https://go.dev/doc/install). Current [LocalStack](https://docs.localstack.cloud/getting-started/installation/)
images require `LOCALSTACK_AUTH_TOKEN`. The runner does not start or stop the
service defined in [docker-compose.yml](./docker-compose.yml).

```sh
export LOCALSTACK_AUTH_TOKEN=...
docker compose up -d localstack
TEST_BACKEND=localstack ./scripts/test-infra.sh
# Or run directly: (cd infra_tests && go test -v -timeout 30m)
docker compose down
```

### AWS Emulator (opt-in)

Install Go, Terraform, Docker Compose, [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html),
`jq`, and `curl`. No AWS or LocalStack credentials are needed.

```sh
TEST_BACKEND=aws-emulator ./scripts/test-infra.sh
```

The runner builds [AWS Emulator v1.0.556](https://github.com/aws/aws-emulator/releases/tag/v1.0.556),
prints its live HTTPS health and signed STS caller identity, runs both tests, and
tears down its Compose project. Set `AWS_EMULATOR_PORT` or `AWS_EMULATOR_REGION`
to override the manifest. Use distinct ports for concurrent projects or regions:
the host port, container port, and emulator listener port must match. Each run
uses one account and one selected region.

The emulator generates `config`, `emulator_credentials`, and `ca.pem` in
`$HOME/.aws/emulator/<port>/`. The runner uses those files for authentication and
TLS verification in that run only; it does not change global AWS configuration
or trust. Each test gets a fresh Terraform working directory. No emulator resource
state persists; this setup is not for production or load testing.

### Backend gaps

No suite-blocking gaps were observed on AWS Emulator v1.0.556. Both tests
passed on macOS arm64 Docker Desktop with the emulator and with a fresh
LocalStack project. The emulator passed signed STS `GetCallerIdentity`, S3
versioning, policy presence and logging checks; Terraform apply and destroy
completed for both tests on both backends. No fallback was needed.

The [release scores](https://github.com/aws/aws-emulator/releases/download/v1.0.556/scores.json.gz)
count units mentioning each service (including cross-service units), not this fixture:

| Service | Pass | Fail | Skipped |
| --- | ---: | ---: | ---: |
| S3 | 2,483 | 488 | 17 |
| STS | 222 | 65 | 7 |

AWS Emulator's [virtual-hosted S3 gap](https://github.com/aws/aws-emulator/blob/v1.0.556/issues/things-that-do-not-work/20261029T083133Z-s3-virtual-hosted-style-bucket-addressing-is-unreachable.md)
is why these tests use path style on its IP endpoint. Use LocalStack for
virtual-host-specific tests. The upstream [S3 `GetBucketPolicy` 404 issue](https://github.com/aws/aws-emulator/blob/v1.0.556/issues/things-that-do-not-work/20261103T071133Z-s3-getbucketpolicy-answers-404-xml-for-a-bucket.md)
uses a different policy fixture; it is not evidence of a failure here.

These tests check policy presence, not enforcement. Use isolated real AWS for
IAM and resource-policy enforcement tests, and use LocalStack only where it
supports the compute or networking behavior under test; otherwise use real AWS.

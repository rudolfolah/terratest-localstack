# terratest-localstack
Run Terratest with LocalStack

Based on the following documentation and code:
- ["Terraform (IaC) testing: Localstack + Terratest.", Corentin Le Devedec](https://medium.com/@ledevedeccorentin/terraform-iac-testing-localstack-terratest-9946dafe98b6)
  - [ldcorentin/aws-terratest-localstack](https://github.com/ldcorentin/aws-terratest-localstack)
- [edsoncelio/terratest-localstack](https://github.com/edsoncelio/terratest-localstack)
- [icarrera/terratest-localstack-example](https://github.com/icarrera/terratest-localstack-example)

## Requirements
- [Docker](https://docs.docker.com/get-docker/)
- [Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli)
- [LocalStack](https://docs.localstack.cloud/getting-started/installation/)
  - [LocalStack configuration for Terraform](https://docs.localstack.cloud/user-guide/integrations/terraform/)
  - `LOCALSTACK_AUTH_TOKEN` is required by current LocalStack Docker images, including the free Hobby tier.
- [Go](https://golang.org/doc/install)
  - [Terratest](https://terratest.gruntwork.io/docs/getting-started/quick-start/)

## Terraform
The [provider is configured to use LocalStack](./infra/provider.tf) by setting up the endpoint URLs to point to localhost and the port where LocalStack is running the service.

Note that the `s3_use_path_style` parameter can be set to `true` if the S3 endpoint is set to `http://localhost:4566`. The documentation notes this may happen if there are DNS resolution issues when using `http://s3.localhost.localstack.cloud:4566` as the S3 endpoint.

The infrastructure is based on the examples from terratest:
- [terraform-aws-s3-example](https://github.com/gruntwork-io/terratest/tree/master/examples/terraform-aws-s3-example)
- [terraform-aws-rds-example](https://github.com/gruntwork-io/terratest/tree/master/examples/terraform-aws-rds-example)

## Localstack
The [Docker Compose file](./docker-compose.yml) is configured to run LocalStack. The compose file can be included in your own project's compose file or used as a reference so that your app code can interact with LocalStack services.

LocalStack can be run as a Docker container.

## Terratest
Terratest is able to use localstack for creating and destroying resources. The provider overrides that set the endpoints to point to Localstack are used there.

However, Terratest's assertions use the AWS SDK to interact with resources.

There are two supported paths in the tests:
1. Terratest AWS assertions use the AWS SDK for Go v2 endpoint environment variable:
   - `AWS_ENDPOINT_URL_S3=http://s3.localhost.localstack.cloud:4566`
   - static `test` credentials
2. The custom S3 client test builds an AWS SDK for Go v2 client with `BaseEndpoint` set to the same LocalStack S3 endpoint.

The previous Terratest fork and Go module `replace` directive are no longer needed.

# Running the tests

```shell
# Required by current LocalStack images.
export LOCALSTACK_AUTH_TOKEN=...

# Start LocalStack.
docker compose up -d localstack

# Optional smoke check.
curl http://localhost:4566/_localstack/health

# Run the tests.
cd infra_tests
go test -v -timeout 30m
```

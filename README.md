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
- [Go](https://golang.org/doc/install)
  - [Terratest](https://terratest.gruntwork.io/docs/getting-started/quick-start/)

## Terraform
The [provider is configured to use LocalStack](./infra/provider.tf) by setting up the endpoint URLs to point to localhost and the port where LocalStack is running the service.

Note that the `s3_force_path_style` parameter can be set to `true` if the S3 endpoint is set to `http://localhost:4566`. The documentation notes this may happen if there are DNS resolution issues when using `http://s3.localhost.localstack.cloud:4566` as the S3 endpoint.

## Localstack
The [Docker Compose file](./docker-compose.yml) is configured to run LocalStack. The compose file can be included in your own project's compose file or used as a reference so that your app code can interact with LocalStack services.

LocalStack can be run as a Docker container.

# Running the tests

```shell
# Start localstack (detached)
docker run -d -it -p 4566:4566 -p 4510-4559:4510-4559 localstack/localstack
```
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

There are a few ways around this:
1. using a fork of Terratest that allows global override of the AWS endpoints
   - The base endpoint can be used to override the endpoint when creating a service client.
   - The endpoint resolver can be used to override the endpoint for all service clients.
   - In the go modules, the `replace` directive points to the fork
2. creating a custom client for the resource and using that for assertions
    - The custom client is created with the endpoint set to LocalStack
    - This needs to be done for each resource that is being tested
    - Terratest is used for setting up and tear down rather than for assertions

Both methods are demonstrated in the tests.

# Running the tests

```shell
# Create a network for localstack
docker network create my-network

# Start localstack (detached)
docker run -d -it --name localstack_main -p 4566:4566 -p 4510-4559:4510-4559 --network my-network -e MAIN_DOCKER_NETWORK=my-network localstack/localstack

export AWS_ENDPOINT_URL=http://localhost:4567
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# Ensure it's running correctly, these should return the same result since they use the same endpoint url
aws s3api list-buckets
aws --endpoint-url http://localhost:4566 s3api list-buckets

# Run the tests
cd infra_test
go test -v -timeout 30m
```

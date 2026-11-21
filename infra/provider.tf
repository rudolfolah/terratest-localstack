locals {
  localstack           = jsondecode(file("${path.module}/test-backends.json")).localstack
  aws_gateway_endpoint = coalesce(var.aws_emulator_endpoint, local.localstack.gateway)
}

provider "aws" {
  region     = var.region
  access_key = var.aws_emulator_endpoint == null ? local.localstack.access_key : null
  secret_key = var.aws_emulator_endpoint == null ? local.localstack.secret_key : null

  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  s3_use_path_style = var.aws_emulator_endpoint != null

  endpoints {
    apigateway     = local.aws_gateway_endpoint
    apigatewayv2   = local.aws_gateway_endpoint
    cloudformation = local.aws_gateway_endpoint
    cloudwatch     = local.aws_gateway_endpoint
    dynamodb       = local.aws_gateway_endpoint
    ec2            = local.aws_gateway_endpoint
    es             = local.aws_gateway_endpoint
    elasticache    = local.aws_gateway_endpoint
    firehose       = local.aws_gateway_endpoint
    iam            = local.aws_gateway_endpoint
    kinesis        = local.aws_gateway_endpoint
    lambda         = local.aws_gateway_endpoint
    rds            = local.aws_gateway_endpoint
    redshift       = local.aws_gateway_endpoint
    route53        = local.aws_gateway_endpoint
    s3             = coalesce(var.aws_emulator_endpoint, local.localstack.s3)
    secretsmanager = local.aws_gateway_endpoint
    ses            = local.aws_gateway_endpoint
    sns            = local.aws_gateway_endpoint
    sqs            = local.aws_gateway_endpoint
    ssm            = local.aws_gateway_endpoint
    stepfunctions  = local.aws_gateway_endpoint
    sts            = local.aws_gateway_endpoint
  }
}

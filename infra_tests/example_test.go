package infra_tests

import (
	"fmt"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	terraaws "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

const localStackS3Endpoint = "http://s3.localhost.localstack.cloud:4566"

// Original source code: https://github.com/gruntwork-io/terratest/blob/master/test/terraform_aws_s3_example_test.go
// An example of how to test the Terraform module in examples/terraform-aws-s3-example using Terratest.
func TestTerraformAwsS3ExampleWithTerratestAWS(t *testing.T) {
	ctx := t.Context()
	configureLocalStackAWS(t)

	// Give this S3 Bucket a unique ID for a name tag so we can distinguish it from any other Buckets provisioned
	// in your AWS account
	expectedName := fmt.Sprintf("terratest-aws-s3-example-%s", strings.ToLower(random.UniqueId()))

	// Give this S3 Bucket an environment to operate as a part of for the purposes of resource tagging
	expectedEnvironment := "Automated Testing"

	// Pick a random AWS region to test in. This helps ensure your code works in all regions.
	awsRegion := terraaws.GetRandomStableRegion(t, nil, nil)

	// Construct the terraform options with default retryable errors to handle the most common retryable errors in
	// terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: "../infra",

		// Variables to pass to our Terraform code using -var options
		Vars: map[string]interface{}{
			"tag_bucket_name":        expectedName,
			"tag_bucket_environment": expectedEnvironment,
			"with_policy":            "true",
			"region":                 awsRegion,
		},
	})

	// At the end of the test, run `terraform destroy` to clean up any resources that were created
	defer terraform.DestroyContext(t, ctx, terraformOptions)

	// This will run `terraform init` and `terraform apply` and fail the test if there are any errors
	terraform.InitAndApplyContext(t, ctx, terraformOptions)

	// Run `terraform output` to get the value of an output variable
	bucketID := terraform.OutputContext(t, ctx, terraformOptions, "bucket_id")

	// Verify that our Bucket has versioning enabled
	actualStatus := terraaws.GetS3BucketVersioningContext(t, ctx, awsRegion, bucketID)
	expectedStatus := "Enabled"
	assert.Equal(t, expectedStatus, actualStatus)

	// Verify that our Bucket has a policy attached
	terraaws.AssertS3BucketPolicyExistsContext(t, ctx, awsRegion, bucketID)

	// Verify that our bucket has server access logging TargetBucket set to what's expected
	loggingTargetBucket := terraaws.GetS3BucketLoggingTargetContext(t, ctx, awsRegion, bucketID)
	expectedLogsTargetBucket := fmt.Sprintf("%s-logs", bucketID)
	loggingObjectTargetPrefix := terraaws.GetS3BucketLoggingTargetPrefixContext(t, ctx, awsRegion, bucketID)
	expectedLogsTargetPrefix := "TFStateLogs/"

	assert.Equal(t, expectedLogsTargetBucket, loggingTargetBucket)
	assert.Equal(t, expectedLogsTargetPrefix, loggingObjectTargetPrefix)
}

func TestTerraformAwsS3ExampleWithCustomSession(t *testing.T) {
	ctx := t.Context()
	configureLocalStackAWS(t)

	expectedName := fmt.Sprintf("terratest-aws-s3-example-%s", strings.ToLower(random.UniqueId()))
	expectedEnvironment := "Automated Testing"

	awsRegion := terraaws.GetRandomStableRegion(t, nil, nil)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(awsRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Fatal(err)
	}
	s3Client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = awssdk.String(localStackS3Endpoint)
	})

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../infra",
		Vars: map[string]interface{}{
			"tag_bucket_name":        expectedName,
			"tag_bucket_environment": expectedEnvironment,
			"with_policy":            "true",
			"region":                 awsRegion,
		},
	})

	defer terraform.DestroyContext(t, ctx, terraformOptions)

	terraform.InitAndApplyContext(t, ctx, terraformOptions)

	bucketID := terraform.OutputContext(t, ctx, terraformOptions, "bucket_id")

	{
		// Based on: https://github.com/ldcorentin/aws-terratest-localstack/blob/main/test/terratest/terraform_test.go
		response, err := s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
			Bucket: awssdk.String(bucketID),
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, types.BucketVersioningStatusEnabled, response.Status)
	}

	{
		policy, err := s3Client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
			Bucket: awssdk.String(bucketID),
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, policy.Policy)
	}

	{
		response, err := s3Client.GetBucketLogging(ctx, &s3.GetBucketLoggingInput{
			Bucket: awssdk.String(bucketID),
		})
		if err != nil {
			t.Fatal(err)
		}
		if response.LoggingEnabled == nil {
			t.Fatal("expected bucket logging to be enabled")
		}
		loggingTargetBucket := awssdk.ToString(response.LoggingEnabled.TargetBucket)
		expectedLogsTargetBucket := fmt.Sprintf("%s-logs", bucketID)
		loggingObjectTargetPrefix := awssdk.ToString(response.LoggingEnabled.TargetPrefix)
		expectedLogsTargetPrefix := "TFStateLogs/"

		assert.Equal(t, expectedLogsTargetBucket, loggingTargetBucket)
		assert.Equal(t, expectedLogsTargetPrefix, loggingObjectTargetPrefix)
	}
}

func configureLocalStackAWS(t *testing.T) {
	t.Helper()

	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ENDPOINT_URL_S3", localStackS3Endpoint)
}

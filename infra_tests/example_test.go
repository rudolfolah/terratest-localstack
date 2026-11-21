package infra_tests

import (
	"fmt"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	terraaws "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// Original source code: https://github.com/gruntwork-io/terratest/blob/master/test/terraform_aws_s3_example_test.go
// An example of how to test the Terraform module in examples/terraform-aws-s3-example using Terratest.
func TestTerraformAwsS3ExampleWithTerratestAWS(t *testing.T) {
	ctx := t.Context()
	backend := configureAWSBackend(t)

	// Give this S3 Bucket a unique ID for a name tag so we can distinguish it from any other Buckets provisioned
	// in your AWS account
	expectedName := fmt.Sprintf("terratest-aws-s3-example-%s", strings.ToLower(random.UniqueId()))

	// Give this S3 Bucket an environment to operate as a part of for the purposes of resource tagging
	expectedEnvironment := "Automated Testing"

	awsRegion := backend.region

	terraformOptions := backend.terraformOptions(t, expectedName, expectedEnvironment)

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
	backend := configureAWSBackend(t)

	expectedName := fmt.Sprintf("terratest-aws-s3-example-%s", strings.ToLower(random.UniqueId()))
	expectedEnvironment := "Automated Testing"

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(backend.region))
	if err != nil {
		t.Fatal(err)
	}
	s3Client := s3.NewFromConfig(cfg)

	terraformOptions := backend.terraformOptions(t, expectedName, expectedEnvironment)

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

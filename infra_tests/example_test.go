package infra_tests

import (
	"fmt"
	"strings"
	"testing"

	aws_sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	aws_session "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// Original source code: https://github.com/gruntwork-io/terratest/blob/master/test/terraform_aws_s3_example_test.go
// An example of how to test the Terraform module in examples/terraform-aws-s3-example using Terratest.
func TestTerraformAwsS3ExampleWithResolver(t *testing.T) {
	resolver := endpoints.ResolverFunc(func(service, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		return endpoints.ResolvedEndpoint{
			URL:           "http://localhost:4566",
			SigningRegion: "custom-signing-region",
		}, nil
	})
	aws.SetBaseAWSConfig(aws_sdk.NewConfig().WithEndpointResolver(resolver))

	// Give this S3 Bucket a unique ID for a name tag so we can distinguish it from any other Buckets provisioned
	// in your AWS account
	expectedName := fmt.Sprintf("terratest-aws-s3-example-%s", strings.ToLower(random.UniqueId()))

	// Give this S3 Bucket an environment to operate as a part of for the purposes of resource tagging
	expectedEnvironment := "Automated Testing"

	// Pick a random AWS region to test in. This helps ensure your code works in all regions.
	awsRegion := aws.GetRandomStableRegion(t, nil, nil)

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
	defer terraform.Destroy(t, terraformOptions)

	// This will run `terraform init` and `terraform apply` and fail the test if there are any errors
	terraform.InitAndApply(t, terraformOptions)

	// Run `terraform output` to get the value of an output variable
	bucketID := terraform.Output(t, terraformOptions, "bucket_id")

	// Verify that our Bucket has versioning enabled
	actualStatus := aws.GetS3BucketVersioning(t, awsRegion, bucketID)
	expectedStatus := "Enabled"
	assert.Equal(t, expectedStatus, actualStatus)

	// Verify that our Bucket has a policy attached
	aws.AssertS3BucketPolicyExists(t, awsRegion, bucketID)

	// Verify that our bucket has server access logging TargetBucket set to what's expected
	loggingTargetBucket := aws.GetS3BucketLoggingTarget(t, awsRegion, bucketID)
	expectedLogsTargetBucket := fmt.Sprintf("%s-logs", bucketID)
	loggingObjectTargetPrefix := aws.GetS3BucketLoggingTargetPrefix(t, awsRegion, bucketID)
	expectedLogsTargetPrefix := "TFStateLogs/"

	assert.Equal(t, expectedLogsTargetBucket, loggingTargetBucket)
	assert.Equal(t, expectedLogsTargetPrefix, loggingObjectTargetPrefix)
}

func TestTerraformAwsS3ExampleWithCustomSession(t *testing.T) {
	expectedName := fmt.Sprintf("terratest-aws-s3-example-%s", strings.ToLower(random.UniqueId()))
	expectedEnvironment := "Automated Testing"

	awsRegion := aws.GetRandomStableRegion(t, nil, nil)

	// Based on: https://github.com/ldcorentin/aws-terratest-localstack/blob/main/test/terratest/terraform_test.go
	session := s3.New(aws_session.Must(aws_session.NewSession(&aws_sdk.Config{
		Region:           aws_sdk.String(awsRegion),
		Endpoint:         aws_sdk.String("http://s3.localhost.localstack.cloud:4566"),
		S3ForcePathStyle: aws_sdk.Bool(false),
		//Endpoint: aws_sdk.String("http://localhost:4566"),
		//S3ForcePathStyle: aws_sdk.Bool(true),
	})))

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../infra",
		Vars: map[string]interface{}{
			"tag_bucket_name":        expectedName,
			"tag_bucket_environment": expectedEnvironment,
			"with_policy":            "true",
			"region":                 awsRegion,
		},
	})

	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	bucketID := terraform.Output(t, terraformOptions, "bucket_id")

	{
		// Based on: https://github.com/ldcorentin/aws-terratest-localstack/blob/main/test/terratest/terraform_test.go
		response, err := session.GetBucketVersioning(&s3.GetBucketVersioningInput{
			Bucket: aws_sdk.String(bucketID),
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "Enabled", *response.Status)
	}

	{
		policy, err := session.GetBucketPolicy(&s3.GetBucketPolicyInput{
			Bucket: aws_sdk.String(bucketID),
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, policy)
	}

	{
		response, err := session.GetBucketLogging(&s3.GetBucketLoggingInput{
			Bucket: aws_sdk.String(bucketID),
		})
		if err != nil {
			t.Fatal(err)
		}
		loggingTargetBucket := aws_sdk.StringValue(response.LoggingEnabled.TargetBucket)
		expectedLogsTargetBucket := fmt.Sprintf("%s-logs", bucketID)
		loggingObjectTargetPrefix := aws_sdk.StringValue(response.LoggingEnabled.TargetPrefix)
		expectedLogsTargetPrefix := "TFStateLogs/"

		assert.Equal(t, expectedLogsTargetBucket, loggingTargetBucket)
		assert.Equal(t, expectedLogsTargetPrefix, loggingObjectTargetPrefix)
	}
	fmt.Println("It works!")
}

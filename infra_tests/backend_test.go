package infra_tests

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	terraaws "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

type backendConfig struct {
	region, terraformDir, emulatorEndpoint string
}

func configureAWSBackend(t *testing.T) backendConfig {
	t.Helper()

	const terraformDir = "../infra"
	const manifest = "test-backends.json"
	var backend backendConfig
	switch selector := os.Getenv("TEST_BACKEND"); selector {
	case "", "localstack":
		path := filepath.Join(terraformDir, manifest)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var defaults struct {
			LocalStack struct {
				S3        string `json:"s3"`
				AccessKey string `json:"access_key"`
				SecretKey string `json:"secret_key"`
			} `json:"localstack"`
		}
		if err := json.Unmarshal(data, &defaults); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		localstack := defaults.LocalStack
		if localstack.S3 == "" || localstack.AccessKey == "" || localstack.SecretKey == "" {
			t.Fatalf("%s: localstack requires s3, access_key and secret_key", path)
		}
		t.Setenv("AWS_ACCESS_KEY_ID", localstack.AccessKey)
		t.Setenv("AWS_SECRET_ACCESS_KEY", localstack.SecretKey)
		t.Setenv("AWS_SESSION_TOKEN", "")
		t.Setenv("AWS_ENDPOINT_URL_S3", localstack.S3)
		t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
		backend.region = terraaws.GetRandomStableRegion(t, nil, nil)
		backend.terraformDir = terraformDir
	case "aws-emulator":
		for _, name := range []string{"AWS_REGION", "AWS_ENDPOINT_URL", "AWS_CONFIG_FILE", "AWS_SHARED_CREDENTIALS_FILE", "AWS_CA_BUNDLE"} {
			if os.Getenv(name) == "" {
				t.Fatalf("missing %s; run TEST_BACKEND=aws-emulator ./scripts/test-infra.sh", name)
			}
		}
		backend.region = os.Getenv("AWS_REGION")
		backend.emulatorEndpoint = os.Getenv("AWS_ENDPOINT_URL")
		endpoint, err := url.Parse(backend.emulatorEndpoint)
		if err != nil || endpoint.Scheme != "https" || endpoint.Host != "127.0.0.1:"+endpoint.Port() ||
			endpoint.User != nil || endpoint.Path != "" || endpoint.ForceQuery || endpoint.RawQuery != "" ||
			strings.Contains(backend.emulatorEndpoint, "#") {
			t.Fatal("AWS_ENDPOINT_URL must be https://127.0.0.1:<port>")
		}
		port, err := strconv.Atoi(endpoint.Port())
		if err != nil || port < 1 || port > 65535 {
			t.Fatal("AWS_ENDPOINT_URL must specify a port from 1 to 65535")
		}

		files, err := filepath.Glob(filepath.Join(terraformDir, "*.tf"))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, filepath.Join(terraformDir, ".terraform.lock.hcl"), filepath.Join(terraformDir, manifest))
		backend.terraformDir = t.TempDir()
		for _, path := range files {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(backend.terraformDir, filepath.Base(path)), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	default:
		t.Fatalf("invalid TEST_BACKEND %q (want localstack or aws-emulator)", selector)
	}
	return backend
}

func (b backendConfig) terraformOptions(t *testing.T, name, environment string) *terraform.Options {
	t.Helper()

	vars := map[string]interface{}{
		"tag_bucket_name":        name,
		"tag_bucket_environment": environment,
		"with_policy":            "true",
		"region":                 b.region,
	}
	if b.emulatorEndpoint != "" {
		vars["aws_emulator_endpoint"] = b.emulatorEndpoint
	}
	return terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: b.terraformDir,
		Vars:         vars,
	})
}

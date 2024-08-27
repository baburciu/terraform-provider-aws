// Code generated by internal/generate/serviceendpointtests/main.go; DO NOT EDIT.

package signer_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	aws_sdkv2 "github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	signer_sdkv2 "github.com/aws/aws-sdk-go-v2/service/signer"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/aws-sdk-go-base/v2/servicemocks"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	terraformsdk "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/provider"
	"github.com/hashicorp/terraform-provider-aws/names"
)

type endpointTestCase struct {
	with     []setupFunc
	expected caseExpectations
}

type caseSetup struct {
	config               map[string]any
	configFile           configFile
	environmentVariables map[string]string
}

type configFile struct {
	baseUrl    string
	serviceUrl string
}

type caseExpectations struct {
	diags    diag.Diagnostics
	endpoint string
	region   string
}

type apiCallParams struct {
	endpoint string
	region   string
}

type setupFunc func(setup *caseSetup)

type callFunc func(ctx context.Context, t *testing.T, meta *conns.AWSClient) apiCallParams

const (
	packageNameConfigEndpoint = "https://packagename-config.endpoint.test/"
	awsServiceEnvvarEndpoint  = "https://service-envvar.endpoint.test/"
	baseEnvvarEndpoint        = "https://base-envvar.endpoint.test/"
	serviceConfigFileEndpoint = "https://service-configfile.endpoint.test/"
	baseConfigFileEndpoint    = "https://base-configfile.endpoint.test/"
)

const (
	packageName = "signer"
	awsEnvVar   = "AWS_ENDPOINT_URL_SIGNER"
	baseEnvVar  = "AWS_ENDPOINT_URL"
	configParam = "signer"
)

const (
	expectedCallRegion = "us-west-2" //lintignore:AWSAT003
)

func TestEndpointConfiguration(t *testing.T) { //nolint:paralleltest // uses t.Setenv
	const providerRegion = "us-west-2" //lintignore:AWSAT003
	const expectedEndpointRegion = providerRegion

	testcases := map[string]endpointTestCase{
		"no config": {
			with:     []setupFunc{withNoConfig},
			expected: expectDefaultEndpoint(t, expectedEndpointRegion),
		},

		// Package name endpoint on Config

		"package name endpoint config": {
			with: []setupFunc{
				withPackageNameEndpointInConfig,
			},
			expected: expectPackageNameConfigEndpoint(),
		},

		"package name endpoint config overrides aws service envvar": {
			with: []setupFunc{
				withPackageNameEndpointInConfig,
				withAwsEnvVar,
			},
			expected: expectPackageNameConfigEndpoint(),
		},

		"package name endpoint config overrides base envvar": {
			with: []setupFunc{
				withPackageNameEndpointInConfig,
				withBaseEnvVar,
			},
			expected: expectPackageNameConfigEndpoint(),
		},

		"package name endpoint config overrides service config file": {
			with: []setupFunc{
				withPackageNameEndpointInConfig,
				withServiceEndpointInConfigFile,
			},
			expected: expectPackageNameConfigEndpoint(),
		},

		"package name endpoint config overrides base config file": {
			with: []setupFunc{
				withPackageNameEndpointInConfig,
				withBaseEndpointInConfigFile,
			},
			expected: expectPackageNameConfigEndpoint(),
		},

		// Service endpoint in AWS envvar

		"service aws envvar": {
			with: []setupFunc{
				withAwsEnvVar,
			},
			expected: expectAwsEnvVarEndpoint(),
		},

		"service aws envvar overrides base envvar": {
			with: []setupFunc{
				withAwsEnvVar,
				withBaseEnvVar,
			},
			expected: expectAwsEnvVarEndpoint(),
		},

		"service aws envvar overrides service config file": {
			with: []setupFunc{
				withAwsEnvVar,
				withServiceEndpointInConfigFile,
			},
			expected: expectAwsEnvVarEndpoint(),
		},

		"service aws envvar overrides base config file": {
			with: []setupFunc{
				withAwsEnvVar,
				withBaseEndpointInConfigFile,
			},
			expected: expectAwsEnvVarEndpoint(),
		},

		// Base endpoint in envvar

		"base endpoint envvar": {
			with: []setupFunc{
				withBaseEnvVar,
			},
			expected: expectBaseEnvVarEndpoint(),
		},

		"base endpoint envvar overrides service config file": {
			with: []setupFunc{
				withBaseEnvVar,
				withServiceEndpointInConfigFile,
			},
			expected: expectBaseEnvVarEndpoint(),
		},

		"base endpoint envvar overrides base config file": {
			with: []setupFunc{
				withBaseEnvVar,
				withBaseEndpointInConfigFile,
			},
			expected: expectBaseEnvVarEndpoint(),
		},

		// Service endpoint in config file

		"service config file": {
			with: []setupFunc{
				withServiceEndpointInConfigFile,
			},
			expected: expectServiceConfigFileEndpoint(),
		},

		"service config file overrides base config file": {
			with: []setupFunc{
				withServiceEndpointInConfigFile,
				withBaseEndpointInConfigFile,
			},
			expected: expectServiceConfigFileEndpoint(),
		},

		// Base endpoint in config file

		"base endpoint config file": {
			with: []setupFunc{
				withBaseEndpointInConfigFile,
			},
			expected: expectBaseConfigFileEndpoint(),
		},

		// Use FIPS endpoint on Config

		"use fips config": {
			with: []setupFunc{
				withUseFIPSInConfig,
			},
			expected: expectDefaultFIPSEndpoint(t, expectedEndpointRegion),
		},

		"use fips config with package name endpoint config": {
			with: []setupFunc{
				withUseFIPSInConfig,
				withPackageNameEndpointInConfig,
			},
			expected: expectPackageNameConfigEndpoint(),
		},
	}

	for name, testcase := range testcases { //nolint:paralleltest // uses t.Setenv
		t.Run(name, func(t *testing.T) {
			testEndpointCase(t, providerRegion, testcase, callService)
		})
	}
}

func defaultEndpoint(region string) (url.URL, error) {
	r := signer_sdkv2.NewDefaultEndpointResolverV2()

	ep, err := r.ResolveEndpoint(context.Background(), signer_sdkv2.EndpointParameters{
		Region: aws_sdkv2.String(region),
	})
	if err != nil {
		return url.URL{}, err
	}

	if ep.URI.Path == "" {
		ep.URI.Path = "/"
	}

	return ep.URI, nil
}

func defaultFIPSEndpoint(region string) (url.URL, error) {
	r := signer_sdkv2.NewDefaultEndpointResolverV2()

	ep, err := r.ResolveEndpoint(context.Background(), signer_sdkv2.EndpointParameters{
		Region:  aws_sdkv2.String(region),
		UseFIPS: aws_sdkv2.Bool(true),
	})
	if err != nil {
		return url.URL{}, err
	}

	if ep.URI.Path == "" {
		ep.URI.Path = "/"
	}

	return ep.URI, nil
}

func callService(ctx context.Context, t *testing.T, meta *conns.AWSClient) apiCallParams {
	t.Helper()

	client := meta.SignerClient(ctx)

	var result apiCallParams

	_, err := client.ListSigningJobs(ctx, &signer_sdkv2.ListSigningJobsInput{},
		func(opts *signer_sdkv2.Options) {
			opts.APIOptions = append(opts.APIOptions,
				addRetrieveEndpointURLMiddleware(t, &result.endpoint),
				addRetrieveRegionMiddleware(&result.region),
				addCancelRequestMiddleware(),
			)
		},
	)
	if err == nil {
		t.Fatal("Expected an error, got none")
	} else if !errors.Is(err, errCancelOperation) {
		t.Fatalf("Unexpected error: %s", err)
	}

	return result
}

func withNoConfig(_ *caseSetup) {
	// no-op
}

func withPackageNameEndpointInConfig(setup *caseSetup) {
	if _, ok := setup.config[names.AttrEndpoints]; !ok {
		setup.config[names.AttrEndpoints] = []any{
			map[string]any{},
		}
	}
	endpoints := setup.config[names.AttrEndpoints].([]any)[0].(map[string]any)
	endpoints[packageName] = packageNameConfigEndpoint
}

func withAwsEnvVar(setup *caseSetup) {
	setup.environmentVariables[awsEnvVar] = awsServiceEnvvarEndpoint
}

func withBaseEnvVar(setup *caseSetup) {
	setup.environmentVariables[baseEnvVar] = baseEnvvarEndpoint
}

func withServiceEndpointInConfigFile(setup *caseSetup) {
	setup.configFile.serviceUrl = serviceConfigFileEndpoint
}

func withBaseEndpointInConfigFile(setup *caseSetup) {
	setup.configFile.baseUrl = baseConfigFileEndpoint
}

func withUseFIPSInConfig(setup *caseSetup) {
	setup.config["use_fips_endpoint"] = true
}

func expectDefaultEndpoint(t *testing.T, region string) caseExpectations {
	t.Helper()

	endpoint, err := defaultEndpoint(region)
	if err != nil {
		t.Fatalf("resolving accessanalyzer default endpoint: %s", err)
	}

	return caseExpectations{
		endpoint: endpoint.String(),
		region:   expectedCallRegion,
	}
}

func expectDefaultFIPSEndpoint(t *testing.T, region string) caseExpectations {
	t.Helper()

	endpoint, err := defaultFIPSEndpoint(region)
	if err != nil {
		t.Fatalf("resolving accessanalyzer FIPS endpoint: %s", err)
	}

	hostname := endpoint.Hostname()
	_, err = net.LookupHost(hostname)
	if dnsErr, ok := errs.As[*net.DNSError](err); ok && dnsErr.IsNotFound {
		return expectDefaultEndpoint(t, region)
	} else if err != nil {
		t.Fatalf("looking up accessanalyzer endpoint %q: %s", hostname, err)
	}

	return caseExpectations{
		endpoint: endpoint.String(),
		region:   expectedCallRegion,
	}
}

func expectPackageNameConfigEndpoint() caseExpectations {
	return caseExpectations{
		endpoint: packageNameConfigEndpoint,
		region:   expectedCallRegion,
	}
}

func expectAwsEnvVarEndpoint() caseExpectations {
	return caseExpectations{
		endpoint: awsServiceEnvvarEndpoint,
		region:   expectedCallRegion,
	}
}

func expectBaseEnvVarEndpoint() caseExpectations {
	return caseExpectations{
		endpoint: baseEnvvarEndpoint,
		region:   expectedCallRegion,
	}
}

func expectServiceConfigFileEndpoint() caseExpectations {
	return caseExpectations{
		endpoint: serviceConfigFileEndpoint,
		region:   expectedCallRegion,
	}
}

func expectBaseConfigFileEndpoint() caseExpectations {
	return caseExpectations{
		endpoint: baseConfigFileEndpoint,
		region:   expectedCallRegion,
	}
}

func testEndpointCase(t *testing.T, region string, testcase endpointTestCase, callF callFunc) {
	t.Helper()

	ctx := context.Background()

	setup := caseSetup{
		config:               map[string]any{},
		environmentVariables: map[string]string{},
	}

	for _, f := range testcase.with {
		f(&setup)
	}

	config := map[string]any{
		names.AttrAccessKey:                 servicemocks.MockStaticAccessKey,
		names.AttrSecretKey:                 servicemocks.MockStaticSecretKey,
		names.AttrRegion:                    region,
		names.AttrSkipCredentialsValidation: true,
		names.AttrSkipRequestingAccountID:   true,
	}

	maps.Copy(config, setup.config)

	if setup.configFile.baseUrl != "" || setup.configFile.serviceUrl != "" {
		config[names.AttrProfile] = "default"
		tempDir := t.TempDir()
		writeSharedConfigFile(t, &config, tempDir, generateSharedConfigFile(setup.configFile))
	}

	for k, v := range setup.environmentVariables {
		t.Setenv(k, v)
	}

	p, err := provider.New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	expectedDiags := testcase.expected.diags
	expectedDiags = append(
		expectedDiags,
		errs.NewWarningDiagnostic(
			"AWS account ID not found for provider",
			"See https://registry.terraform.io/providers/hashicorp/aws/latest/docs#skip_requesting_account_id for implications.",
		),
	)

	diags := p.Configure(ctx, terraformsdk.NewResourceConfigRaw(config))

	if diff := cmp.Diff(diags, expectedDiags, cmp.Comparer(sdkdiag.Comparer)); diff != "" {
		t.Errorf("unexpected diagnostics difference: %s", diff)
	}

	if diags.HasError() {
		return
	}

	meta := p.Meta().(*conns.AWSClient)

	callParams := callF(ctx, t, meta)

	if e, a := testcase.expected.endpoint, callParams.endpoint; e != a {
		t.Errorf("expected endpoint %q, got %q", e, a)
	}

	if e, a := testcase.expected.region, callParams.region; e != a {
		t.Errorf("expected region %q, got %q", e, a)
	}
}

func addRetrieveEndpointURLMiddleware(t *testing.T, endpoint *string) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			retrieveEndpointURLMiddleware(t, endpoint),
			middleware.After,
		)
	}
}

func retrieveEndpointURLMiddleware(t *testing.T, endpoint *string) middleware.FinalizeMiddleware {
	return middleware.FinalizeMiddlewareFunc(
		"Test: Retrieve Endpoint",
		func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			t.Helper()

			request, ok := in.Request.(*smithyhttp.Request)
			if !ok {
				t.Fatalf("Expected *github.com/aws/smithy-go/transport/http.Request, got %s", fullTypeName(in.Request))
			}

			url := request.URL
			url.RawQuery = ""
			url.Path = "/"

			*endpoint = url.String()

			return next.HandleFinalize(ctx, in)
		})
}

func addRetrieveRegionMiddleware(region *string) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Serialize.Add(
			retrieveRegionMiddleware(region),
			middleware.After,
		)
	}
}

func retrieveRegionMiddleware(region *string) middleware.SerializeMiddleware {
	return middleware.SerializeMiddlewareFunc(
		"Test: Retrieve Region",
		func(ctx context.Context, in middleware.SerializeInput, next middleware.SerializeHandler) (middleware.SerializeOutput, middleware.Metadata, error) {
			*region = awsmiddleware.GetRegion(ctx)

			return next.HandleSerialize(ctx, in)
		},
	)
}

var errCancelOperation = fmt.Errorf("Test: Canceling request")

func addCancelRequestMiddleware() func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			cancelRequestMiddleware(),
			middleware.After,
		)
	}
}

// cancelRequestMiddleware creates a Smithy middleware that intercepts the request before sending and cancels it
func cancelRequestMiddleware() middleware.FinalizeMiddleware {
	return middleware.FinalizeMiddlewareFunc(
		"Test: Cancel Requests",
		func(_ context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			return middleware.FinalizeOutput{}, middleware.Metadata{}, errCancelOperation
		})
}

func fullTypeName(i interface{}) string {
	return fullValueTypeName(reflect.ValueOf(i))
}

func fullValueTypeName(v reflect.Value) string {
	if v.Kind() == reflect.Ptr {
		return "*" + fullValueTypeName(reflect.Indirect(v))
	}

	requestType := v.Type()
	return fmt.Sprintf("%s.%s", requestType.PkgPath(), requestType.Name())
}

func generateSharedConfigFile(config configFile) string {
	var buf strings.Builder

	buf.WriteString(`
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`)
	if config.baseUrl != "" {
		buf.WriteString(fmt.Sprintf("endpoint_url = %s\n", config.baseUrl))
	}

	if config.serviceUrl != "" {
		buf.WriteString(fmt.Sprintf(`
services = endpoint-test

[services endpoint-test]
%[1]s =
  endpoint_url = %[2]s
`, configParam, serviceConfigFileEndpoint))
	}

	return buf.String()
}

func writeSharedConfigFile(t *testing.T, config *map[string]any, tempDir, content string) string {
	t.Helper()

	file, err := os.Create(filepath.Join(tempDir, "aws-sdk-go-base-shared-configuration-file"))
	if err != nil {
		t.Fatalf("creating shared configuration file: %s", err)
	}

	_, err = file.WriteString(content)
	if err != nil {
		t.Fatalf(" writing shared configuration file: %s", err)
	}

	if v, ok := (*config)[names.AttrSharedConfigFiles]; !ok {
		(*config)[names.AttrSharedConfigFiles] = []any{file.Name()}
	} else {
		(*config)[names.AttrSharedConfigFiles] = append(v.([]any), file.Name())
	}

	return file.Name()
}

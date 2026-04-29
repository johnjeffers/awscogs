package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"

	cogpricing "github.com/johnjeffers/awscogs/backend/internal/pricing"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "lambda" {
		runLambdaValidation(os.Args[2:])
		return
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	client := pricing.NewFromConfig(cfg)

	// Test 1: AWSELB with productFamily filter
	fmt.Println("=== Test 1: AWSELB productFamily=Load Balancer-Application, location=US East (N. Virginia) ===")
	testQuery(ctx, client, "AWSELB", []types.Filter{
		{Type: types.FilterTypeTermMatch, Field: aws.String("productFamily"), Value: aws.String("Load Balancer-Application")},
		{Type: types.FilterTypeTermMatch, Field: aws.String("location"), Value: aws.String("US East (N. Virginia)")},
	})

	// Test 2: AWSELB with no productFamily
	fmt.Println("\n=== Test 2: AWSELB location=US East (N. Virginia) only ===")
	testQuery(ctx, client, "AWSELB", []types.Filter{
		{Type: types.FilterTypeTermMatch, Field: aws.String("location"), Value: aws.String("US East (N. Virginia)")},
	})

	// Test 3: AWSELB for us-east-2
	fmt.Println("\n=== Test 3: AWSELB productFamily=Load Balancer-Application, location=US East (Ohio) ===")
	testQuery(ctx, client, "AWSELB", []types.Filter{
		{Type: types.FilterTypeTermMatch, Field: aws.String("productFamily"), Value: aws.String("Load Balancer-Application")},
		{Type: types.FilterTypeTermMatch, Field: aws.String("location"), Value: aws.String("US East (Ohio)")},
	})

	// Test 4: List valid attributes for AWSELB
	fmt.Println("\n=== Test 4: DescribeServices for AWSELB ===")
	descOutput, err := client.DescribeServices(ctx, &pricing.DescribeServicesInput{
		ServiceCode: aws.String("AWSELB"),
		MaxResults:  aws.Int32(1),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "DescribeServices error: %v\n", err)
	} else {
		for _, svc := range descOutput.Services {
			fmt.Printf("ServiceCode: %s\n", *svc.ServiceCode)
			fmt.Printf("AttributeNames: %v\n", svc.AttributeNames)
		}
	}
}

func runLambdaValidation(args []string) {
	fs := flag.NewFlagSet("lambda", flag.ExitOnError)
	regionsFlag := fs.String("regions", "us-east-1,us-west-2,eu-west-1", "comma-separated regions to validate")
	architecturesFlag := fs.String("architectures", "x86_64,arm64", "comma-separated Lambda architectures to validate")
	live := fs.Bool("live", false, "run live AWS Pricing API validation")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse flags: %v\n", err)
		os.Exit(2)
	}

	if !*live && os.Getenv("AWSCOGS_LIVE_AWS_TESTS") != "1" {
		fmt.Fprintln(os.Stderr, "lambda pricing validation calls the live AWS Pricing API; pass --live or set AWSCOGS_LIVE_AWS_TESTS=1")
		os.Exit(2)
	}

	ctx := context.Background()
	provider, err := cogpricing.NewAWSProvider(ctx, 5, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pricing provider: %v\n", err)
		os.Exit(1)
	}

	failures := 0
	for _, region := range splitCSV(*regionsFlag) {
		if strings.HasPrefix(region, "us-gov-") && !envEnabled("AWSCOGS_ENABLE_GOVCLOUD") {
			fmt.Printf("SKIP %s: AWSCOGS_ENABLE_GOVCLOUD is not enabled\n", region)
			continue
		}

		for _, architecture := range splitCSV(*architecturesFlag) {
			details, err := provider.GetLambdaPriceDetails(ctx, region, architecture)
			if err != nil {
				failures++
				fmt.Printf("FAIL %s %s: %v\n", region, architecture, err)
				continue
			}

			fmt.Printf(
				"OK %s %s request=%.10f sku=%s usage=%s gb-second=%.10f sku=%s usage=%s matched=%d\n",
				details.Region,
				details.Architecture,
				details.RequestPrice,
				details.RequestSKU,
				details.RequestUsageType,
				details.GBSecondPrice,
				details.GBSecondSKU,
				details.GBSecondUsageType,
				details.MatchedProductCount,
			)
		}
	}

	if failures > 0 {
		os.Exit(1)
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func envEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func testQuery(ctx context.Context, client *pricing.Client, serviceCode string, filters []types.Filter) {
	output, err := client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String(serviceCode),
		Filters:     filters,
		MaxResults:  aws.Int32(20),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR: %v\n", err)
		return
	}

	fmt.Printf("  Found %d products\n", len(output.PriceList))
	for i, pl := range output.PriceList {
		var product map[string]any
		json.Unmarshal([]byte(pl), &product)
		prod, _ := product["product"].(map[string]any)
		attrs, _ := prod["attributes"].(map[string]any)
		fmt.Printf("  [%d] productFamily=%v usagetype=%v group=%v\n", i, prod["productFamily"], attrs["usagetype"], attrs["group"])
	}
}

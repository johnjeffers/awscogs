package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

func main() {
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

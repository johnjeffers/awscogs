package pricing

import "testing"

func TestLambdaUsageTypeClassification(t *testing.T) {
	if !isLambdaRequestUsage("Request") {
		t.Fatal("expected current regional request usage type to match")
	}
	if !isLambdaRequestUsage("Request-ARM") {
		t.Fatal("expected current ARM regional request usage type to match")
	}
	if !isLambdaRequestUsage("USW2-Request") {
		t.Fatal("expected regional request usage type to match")
	}
	if !isLambdaRequestUsage("USE2-Request-ARM") {
		t.Fatal("expected regional ARM request usage type to match")
	}
	if !isLambdaRequestUsage("USE1-Lambda-Requests") {
		t.Fatal("expected regional request usage type to match")
	}
	if !isLambdaX86GBSecondUsage("USE1-Lambda-GB-Second") {
		t.Fatal("expected x86 GB-second usage type to match")
	}
	if isLambdaX86GBSecondUsage("USE1-Lambda-ARM-GB-Second") {
		t.Fatal("ARM GB-second usage type should not match x86")
	}
	if !isLambdaArmGBSecondUsage("USE1-Lambda-GB-Second-ARM") {
		t.Fatal("expected current ARM GB-second usage type to match")
	}
	if !isLambdaArmGBSecondUsage("USE1-Lambda-ARM-GB-Second") {
		t.Fatal("expected ARM GB-second usage type to match")
	}
	if !shouldSkipLambdaUsageType("USE1-Lambda-Provisioned-Concurrency") {
		t.Fatal("expected provisioned concurrency usage type to be skipped")
	}
	if !shouldSkipLambdaUsageType("Lambda-Managed-Instances-Request") {
		t.Fatal("expected managed instance request usage type to be skipped")
	}
}

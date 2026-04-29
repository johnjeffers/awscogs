package pricing

import "testing"

func TestLambdaUsageTypeClassification(t *testing.T) {
	if !isLambdaRequestUsage("USE1-Lambda-Requests") {
		t.Fatal("expected regional request usage type to match")
	}
	if !isLambdaX86GBSecondUsage("USE1-Lambda-GB-Second") {
		t.Fatal("expected x86 GB-second usage type to match")
	}
	if isLambdaX86GBSecondUsage("USE1-Lambda-ARM-GB-Second") {
		t.Fatal("ARM GB-second usage type should not match x86")
	}
	if !isLambdaArmGBSecondUsage("USE1-Lambda-ARM-GB-Second") {
		t.Fatal("expected ARM GB-second usage type to match")
	}
	if !shouldSkipLambdaUsageType("USE1-Lambda-Provisioned-Concurrency") {
		t.Fatal("expected provisioned concurrency usage type to be skipped")
	}
}

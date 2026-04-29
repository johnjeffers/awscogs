package aws

import "testing"

func TestDefaultAccountsForRegionsUsesRegionPartitions(t *testing.T) {
	accounts := defaultAccountsForRegions([]string{"us-gov-west-1", "us-east-1", "us-gov-east-1"})

	if len(accounts) != 2 {
		t.Fatalf("expected 2 partition accounts, got %d", len(accounts))
	}
	if accounts[0].Partition != "aws-us-gov" {
		t.Fatalf("first partition = %q", accounts[0].Partition)
	}
	if accounts[1].Partition != "aws" {
		t.Fatalf("second partition = %q", accounts[1].Partition)
	}
}

func TestAccountPartitionDerivesFromRoleARN(t *testing.T) {
	account := Account{RoleARN: "arn:aws-us-gov:iam::123456789012:role/Audit"}

	if got := account.AccountPartition(); got != "aws-us-gov" {
		t.Fatalf("AccountPartition() = %q", got)
	}
}

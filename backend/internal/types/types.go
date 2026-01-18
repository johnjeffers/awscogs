package types

// CostValue represents a monetary cost value
type CostValue float64

// EC2Instance represents an EC2 instance with its cost
type EC2Instance struct {
	AccountID    string    `json:"accountId"`
	AccountName  string    `json:"accountName"`
	Region       string    `json:"region"`
	InstanceID   string    `json:"instanceId"`
	Name         string    `json:"name"`
	InstanceType string    `json:"instanceType"`
	State        string    `json:"state"`
	HourlyCost   CostValue `json:"hourlyCost"`
}

// EBSVolume represents an EBS volume with its cost
type EBSVolume struct {
	AccountID   string    `json:"accountId"`
	AccountName string    `json:"accountName"`
	Region      string    `json:"region"`
	VolumeID    string    `json:"volumeId"`
	Name        string    `json:"name"`
	VolumeType  string    `json:"volumeType"`
	Size        int32     `json:"size"` // in GiB
	IOPS        int32     `json:"iops"`
	Throughput  int32     `json:"throughput"` // in MiB/s for gp3
	State       string    `json:"state"`
	HourlyCost  CostValue `json:"hourlyCost"`
}

// RDSInstance represents an RDS instance with its cost
type RDSInstance struct {
	AccountID        string    `json:"accountId"`
	AccountName      string    `json:"accountName"`
	Region           string    `json:"region"`
	DBInstanceID     string    `json:"dbInstanceId"`
	Name             string    `json:"name"`
	Engine           string    `json:"engine"`
	EngineVersion    string    `json:"engineVersion"`
	InstanceClass    string    `json:"instanceClass"`
	MultiAZ          bool      `json:"multiAz"`
	StorageType      string    `json:"storageType"`
	AllocatedStorage int32     `json:"allocatedStorage"` // in GiB
	State            string    `json:"state"`
	HourlyCost       CostValue `json:"hourlyCost"`
}

// ECSService represents an ECS service with its cost
type ECSService struct {
	AccountID    string    `json:"accountId"`
	AccountName  string    `json:"accountName"`
	Region       string    `json:"region"`
	ClusterName  string    `json:"clusterName"`
	ServiceName  string    `json:"serviceName"`
	LaunchType   string    `json:"launchType"` // FARGATE, EC2, EXTERNAL
	DesiredCount int32     `json:"desiredCount"`
	RunningCount int32     `json:"runningCount"`
	State        string    `json:"state"` // ACTIVE, DRAINING, INACTIVE
	HourlyCost   CostValue `json:"hourlyCost"`
}

// EKSCluster represents an EKS cluster with its cost
type EKSCluster struct {
	AccountID   string    `json:"accountId"`
	AccountName string    `json:"accountName"`
	Region      string    `json:"region"`
	ClusterName string    `json:"clusterName"`
	Status      string    `json:"status"`
	Version     string    `json:"version"`
	Platform    string    `json:"platform"` // linux, windows
	HourlyCost  CostValue `json:"hourlyCost"`
}

// LoadBalancer represents an Elastic Load Balancer with its cost
type LoadBalancer struct {
	AccountID   string    `json:"accountId"`
	AccountName string    `json:"accountName"`
	Region      string    `json:"region"`
	Name        string    `json:"name"`
	ARN         string    `json:"arn"`
	Type        string    `json:"type"`   // application, network, classic
	Scheme      string    `json:"scheme"` // internet-facing, internal
	State       string    `json:"state"`
	HourlyCost  CostValue `json:"hourlyCost"`
}

// AccountSummary represents cost summary for an AWS account
type AccountSummary struct {
	AccountID   string    `json:"accountId"`
	AccountName string    `json:"accountName"`
	EC2Count    int       `json:"ec2Count"`
	EBSCount    int       `json:"ebsCount"`
	ECSCount    int       `json:"ecsCount"`
	RDSCount    int       `json:"rdsCount"`
	EKSCount    int       `json:"eksCount"`
	ELBCount    int       `json:"elbCount"`
	TotalCost   CostValue `json:"totalCost"`
}

// RegionSummary represents cost summary for a region
type RegionSummary struct {
	Region    string    `json:"region"`
	EC2Count  int       `json:"ec2Count"`
	EBSCount  int       `json:"ebsCount"`
	ECSCount  int       `json:"ecsCount"`
	RDSCount  int       `json:"rdsCount"`
	EKSCount  int       `json:"eksCount"`
	ELBCount  int       `json:"elbCount"`
	TotalCost CostValue `json:"totalCost"`
}

// CostResponse is the API response for cost data
type CostResponse struct {
	Timestamp     string           `json:"timestamp"`
	TotalCost     CostValue        `json:"totalCost"`
	Currency      string           `json:"currency"`
	Accounts      []AccountSummary `json:"accounts,omitempty"`
	Regions       []RegionSummary  `json:"regions,omitempty"`
	EC2Instances  []EC2Instance    `json:"ec2Instances,omitempty"`
	EBSVolumes    []EBSVolume      `json:"ebsVolumes,omitempty"`
	ECSServices   []ECSService     `json:"ecsServices,omitempty"`
	RDSInstances  []RDSInstance    `json:"rdsInstances,omitempty"`
	EKSClusters   []EKSCluster     `json:"eksClusters,omitempty"`
	LoadBalancers []LoadBalancer   `json:"loadBalancers,omitempty"`
	Filters       AppliedFilters   `json:"filters"`
}

// AppliedFilters shows what filters were applied to the response
type AppliedFilters struct {
	Accounts      []string `json:"accounts,omitempty"`
	Regions       []string `json:"regions,omitempty"`
	ResourceTypes []string `json:"resourceTypes,omitempty"`
}

export interface CostResponse {
  timestamp: string;
  totalCost: number;
  currency: string;
  accounts?: AccountSummary[];
  regions?: RegionSummary[];
  ec2Instances?: EC2Instance[];
  ebsVolumes?: EBSVolume[];
  rdsInstances?: RDSInstance[];
  ecsServices?: ECSService[];
  eksClusters?: EKSCluster[];
  loadBalancers?: LoadBalancer[];
  natGateways?: NATGateway[];
  elasticIps?: ElasticIP[];
  secrets?: Secret[];
  publicIpv4s?: PublicIPv4[];
  filters: AppliedFilters;
}

export interface AccountSummary {
  accountId: string;
  accountName: string;
  ec2Count: number;
  ebsCount: number;
  rdsCount: number;
  ecsCount: number;
  eksCount: number;
  elbCount: number;
  natCount: number;
  eipCount: number;
  secretCount: number;
  publicIpv4Count: number;
  totalCost: number;
}

export interface RegionSummary {
  region: string;
  ec2Count: number;
  ebsCount: number;
  rdsCount: number;
  ecsCount: number;
  eksCount: number;
  elbCount: number;
  natCount: number;
  eipCount: number;
  secretCount: number;
  publicIpv4Count: number;
  totalCost: number;
}

export interface EC2Instance {
  accountId: string;
  accountName: string;
  region: string;
  instanceId: string;
  name: string;
  instanceType: string;
  state: string;
  hourlyCost: number;
}

export interface EBSVolume {
  accountId: string;
  accountName: string;
  region: string;
  volumeId: string;
  name: string;
  volumeType: string;
  size: number;
  iops: number;
  throughput: number;
  state: string;
  hourlyCost: number;
}

export interface RDSInstance {
  accountId: string;
  accountName: string;
  region: string;
  dbInstanceId: string;
  name: string;
  engine: string;
  engineVersion: string;
  instanceClass: string;
  multiAz: boolean;
  storageType: string;
  allocatedStorage: number;
  state: string;
  hourlyCost: number;
}

export interface ECSService {
  accountId: string;
  accountName: string;
  region: string;
  clusterName: string;
  serviceName: string;
  launchType: string;
  desiredCount: number;
  runningCount: number;
  state: string;
  hourlyCost: number;
}

export interface EKSCluster {
  accountId: string;
  accountName: string;
  region: string;
  clusterName: string;
  status: string;
  version: string;
  platform: string;
  hourlyCost: number;
}

export interface LoadBalancer {
  accountId: string;
  accountName: string;
  region: string;
  name: string;
  arn: string;
  type: string;
  scheme: string;
  state: string;
  hourlyCost: number;
}

export interface NATGateway {
  accountId: string;
  accountName: string;
  region: string;
  id: string;
  name: string;
  state: string;
  type: string;
  vpcId: string;
  subnetId: string;
  hourlyCost: number;
}

export interface ElasticIP {
  accountId: string;
  accountName: string;
  region: string;
  allocationId: string;
  publicIp: string;
  name: string;
  associationId: string;
  instanceId: string;
  isAssociated: boolean;
  hourlyCost: number;
}

export interface Secret {
  accountId: string;
  accountName: string;
  region: string;
  name: string;
  arn: string;
  description: string;
  hourlyCost: number;
}

export interface PublicIPv4 {
  accountId: string;
  accountName: string;
  region: string;
  publicIp: string;
  instanceId: string;
  instanceName: string;
  hourlyCost: number;
}

export interface AppliedFilters {
  accounts?: string[];
  regions?: string[];
  resourceTypes?: string[];
}

export interface CostFilters {
  accounts?: string[];
  regions?: string[];
  resources?: string[];
}

export const RESOURCE_TYPES = ['ec2', 'ebs', 'ecs', 'rds', 'eks', 'elb', 'nat', 'eip', 'secrets', 'publicipv4'] as const;
export type ResourceType = typeof RESOURCE_TYPES[number];

export interface ConfigResponse {
  accounts: { id: string; name: string }[];
  regions: string[];
}

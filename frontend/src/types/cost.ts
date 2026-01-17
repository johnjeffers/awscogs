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
  filters: AppliedFilters;
}

export interface AccountSummary {
  accountId: string;
  accountName: string;
  ec2Count: number;
  ebsCount: number;
  rdsCount: number;
  ecsCount: number;
  totalCost: number;
}

export interface RegionSummary {
  region: string;
  ec2Count: number;
  ebsCount: number;
  rdsCount: number;
  ecsCount: number;
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

export const RESOURCE_TYPES = ['ec2', 'ebs', 'ecs', 'rds'] as const;
export type ResourceType = typeof RESOURCE_TYPES[number];

export interface ConfigResponse {
  accounts: { id: string; name: string }[];
  regions: string[];
}

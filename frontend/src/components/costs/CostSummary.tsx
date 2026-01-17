import React from 'react';

interface CostSummaryProps {
  totalCost: number;
  ec2Count: number;
  ebsCount: number;
  ecsCount: number;
  rdsCount: number;
  currency: string;
}

export const CostSummary: React.FC<CostSummaryProps> = ({
  totalCost,
  ec2Count,
  ebsCount,
  ecsCount,
  rdsCount,
  currency,
}) => {
  const formatCost = (cost: number, decimals: number = 4) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency,
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    }).format(cost);
  };

  const dailyCost = totalCost * 24;
  const monthlyCost = totalCost * 730;

  return (
    <div className="mb-6">
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Hourly Cost */}
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <dt className="text-sm font-medium text-gray-500 truncate">
              Hourly Cost
            </dt>
            <dd className="mt-1 text-2xl font-semibold text-gray-900">
              {formatCost(totalCost, 2)}
            </dd>
          </div>
        </div>

        {/* Daily Cost */}
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <dt className="text-sm font-medium text-gray-500 truncate">
              Daily Cost
            </dt>
            <dd className="mt-1 text-2xl font-semibold text-gray-900">
              {formatCost(dailyCost, 2)}
            </dd>
          </div>
        </div>

        {/* Monthly Cost */}
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <dt className="text-sm font-medium text-gray-500 truncate">
              Monthly Cost
            </dt>
            <dd className="mt-1 text-2xl font-semibold text-gray-900">
              {formatCost(monthlyCost, 2)}
            </dd>
          </div>
        </div>

        {/* Resource Count */}
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <dt className="text-sm font-medium text-gray-500 truncate">
              Resources
            </dt>
            <dd className="mt-1 text-2xl font-semibold text-gray-900">
              {ec2Count + ebsCount + ecsCount + rdsCount}
            </dd>
            <dd className="text-xs text-gray-500">
              {ec2Count} EC2 / {ebsCount} EBS / {ecsCount} ECS / {rdsCount} RDS
            </dd>
          </div>
        </div>
      </div>
    </div>
  );
};

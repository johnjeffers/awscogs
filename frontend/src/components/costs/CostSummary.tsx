import React from 'react';

interface CostSummaryProps {
  selectedCost: number;
  totalCost: number;
  selectedCount: number;
  totalCount: number;
  currency: string;
}

export const CostSummary: React.FC<CostSummaryProps> = ({
  selectedCost,
  totalCost,
  selectedCount,
  totalCount,
  currency,
}) => {
  const formatCost = (cost: number, decimals: number = 2) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency,
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    }).format(cost);
  };

  const selectedDaily = selectedCost * 24;
  const totalDaily = totalCost * 24;
  const selectedMonthly = selectedCost * 730;
  const totalMonthly = totalCost * 730;

  const showBoth = selectedCost !== totalCost || selectedCount !== totalCount;

  return (
    <div className="mb-6">
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Hourly Cost */}
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <dt className="text-sm font-medium text-gray-500 truncate">
              Hourly Cost
            </dt>
            {showBoth ? (
              <dd className="mt-1">
                <div className="text-2xl font-semibold text-gray-900">{formatCost(selectedCost)}</div>
                <div className="text-sm text-gray-500">of {formatCost(totalCost)} total</div>
              </dd>
            ) : (
              <dd className="mt-1 text-2xl font-semibold text-gray-900">
                {formatCost(totalCost)}
              </dd>
            )}
          </div>
        </div>

        {/* Daily Cost */}
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <dt className="text-sm font-medium text-gray-500 truncate">
              Daily Cost
            </dt>
            {showBoth ? (
              <dd className="mt-1">
                <div className="text-2xl font-semibold text-gray-900">{formatCost(selectedDaily)}</div>
                <div className="text-sm text-gray-500">of {formatCost(totalDaily)} total</div>
              </dd>
            ) : (
              <dd className="mt-1 text-2xl font-semibold text-gray-900">
                {formatCost(totalDaily)}
              </dd>
            )}
          </div>
        </div>

        {/* Monthly Cost */}
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <dt className="text-sm font-medium text-gray-500 truncate">
              Monthly Cost
            </dt>
            {showBoth ? (
              <dd className="mt-1">
                <div className="text-2xl font-semibold text-gray-900">{formatCost(selectedMonthly)}</div>
                <div className="text-sm text-gray-500">of {formatCost(totalMonthly)} total</div>
              </dd>
            ) : (
              <dd className="mt-1 text-2xl font-semibold text-gray-900">
                {formatCost(totalMonthly)}
              </dd>
            )}
          </div>
        </div>

        {/* Resource Count */}
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <dt className="text-sm font-medium text-gray-500 truncate">
              Resources
            </dt>
            {showBoth ? (
              <dd className="mt-1">
                <div className="text-2xl font-semibold text-gray-900">{selectedCount}</div>
                <div className="text-sm text-gray-500">of {totalCount} total</div>
              </dd>
            ) : (
              <dd className="mt-1 text-2xl font-semibold text-gray-900">
                {totalCount}
              </dd>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

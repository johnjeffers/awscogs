import React from 'react';

interface CostSummaryProps {
  selectedCost: number;
  totalCost: number;
  selectedCount: number;
  totalCount: number;
  currency: string;
  traffic?: {
    window: string;
    selectedRequests: number;
    totalRequests: number;
    selectedBandwidth: number;
    totalBandwidth: number;
  };
}

const formatBytes = (bytes: number | undefined): string => {
  if (!bytes || bytes === 0) return '0 B';
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(value < 10 && i > 0 ? 1 : 0)} ${units[i]}`;
};

const formatVolume = (volume: number | undefined): string => {
  if (!volume || volume === 0) return '0';
  return Math.round(volume).toLocaleString();
};

export const CostSummary: React.FC<CostSummaryProps> = ({
  selectedCost,
  totalCost,
  selectedCount,
  totalCount,
  currency,
  traffic,
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
  const showBothTraffic = traffic && (
    traffic.selectedRequests !== traffic.totalRequests ||
    traffic.selectedBandwidth !== traffic.totalBandwidth
  );

  return (
    <div className="mb-6">
      <div className="flex gap-4">
        {/* Resources - always 20% */}
        <div className="w-1/5 flex-shrink-0 bg-white overflow-hidden shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <dt className="text-sm font-medium text-gray-500 truncate">Resources</dt>
            {showBoth ? (
              <dd className="mt-2">
                <div className="text-lg font-semibold text-gray-900">{selectedCount}</div>
                <div className="text-xs text-gray-500">of {totalCount} total</div>
              </dd>
            ) : (
              <dd className="mt-2 text-lg font-semibold text-gray-900">{totalCount}</dd>
            )}
          </div>
        </div>

        {/* Cost and Traffic split remaining 80% */}
        <div className={`flex-1 grid gap-4 ${traffic ? 'grid-cols-2' : 'grid-cols-1'}`}>
          {/* Cost */}
          <div className="bg-white overflow-hidden shadow rounded-lg">
            <div className="px-4 py-5 sm:p-6">
              <dt className="text-sm font-medium text-gray-500 truncate">Cost</dt>
              <dd className="mt-2 grid grid-cols-3 gap-4">
                <div>
                  <div className="text-xs text-gray-400 uppercase">Hourly</div>
                  <div className="text-lg font-semibold text-gray-900">{formatCost(showBoth ? selectedCost : totalCost)}</div>
                  {showBoth && <div className="text-xs text-gray-500">of {formatCost(totalCost)}</div>}
                </div>
                <div>
                  <div className="text-xs text-gray-400 uppercase">Daily</div>
                  <div className="text-lg font-semibold text-gray-900">{formatCost(showBoth ? selectedDaily : totalDaily)}</div>
                  {showBoth && <div className="text-xs text-gray-500">of {formatCost(totalDaily)}</div>}
                </div>
                <div>
                  <div className="text-xs text-gray-400 uppercase">Monthly</div>
                  <div className="text-lg font-semibold text-gray-900">{formatCost(showBoth ? selectedMonthly : totalMonthly)}</div>
                  {showBoth && <div className="text-xs text-gray-500">of {formatCost(totalMonthly)}</div>}
                </div>
              </dd>
            </div>
          </div>

          {/* Traffic */}
          {traffic && (
            <div className="bg-white overflow-hidden shadow rounded-lg">
              <div className="px-4 py-5 sm:p-6">
                <dt className="text-sm font-medium text-gray-500 truncate">Traffic ({traffic.window})</dt>
                <dd className="mt-2 grid grid-cols-2 gap-4">
                  <div>
                    <div className="text-xs text-gray-400 uppercase">Requests</div>
                    <div className="text-lg font-semibold text-gray-900">
                      {formatVolume(showBothTraffic ? traffic.selectedRequests : traffic.totalRequests)}
                    </div>
                    {showBothTraffic && (
                      <div className="text-xs text-gray-500">of {formatVolume(traffic.totalRequests)}</div>
                    )}
                  </div>
                  <div>
                    <div className="text-xs text-gray-400 uppercase">Bandwidth</div>
                    <div className="text-lg font-semibold text-gray-900">
                      {formatBytes(showBothTraffic ? traffic.selectedBandwidth : traffic.totalBandwidth)}
                    </div>
                    {showBothTraffic && (
                      <div className="text-xs text-gray-500">of {formatBytes(traffic.totalBandwidth)}</div>
                    )}
                  </div>
                </dd>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

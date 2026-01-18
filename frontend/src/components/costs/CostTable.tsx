import React, { useState, useMemo } from 'react';
import type { AccountSummary, RegionSummary, EC2Instance, EBSVolume, ECSService, RDSInstance, EKSCluster, LoadBalancer, NATGateway, ElasticIP, Secret } from '../../types/cost';

interface CostTableProps {
  accounts?: AccountSummary[];
  regions?: RegionSummary[];
  ec2?: EC2Instance[];
  ebs?: EBSVolume[];
  ecs?: ECSService[];
  rds?: RDSInstance[];
  eks?: EKSCluster[];
  elb?: LoadBalancer[];
  nat?: NATGateway[];
  eip?: ElasticIP[];
  secrets?: Secret[];
}

type SortDirection = 'asc' | 'desc';

interface SortConfig {
  key: string;
  direction: SortDirection;
}

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100, 'All'] as const;
type PageSize = typeof PAGE_SIZE_OPTIONS[number];

interface PaginationProps {
  currentPage: number;
  totalItems: number;
  pageSize: PageSize;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: PageSize) => void;
}

const Pagination: React.FC<PaginationProps> = ({ currentPage, totalItems, pageSize, onPageChange, onPageSizeChange }) => {
  const effectivePageSize = pageSize === 'All' ? totalItems : pageSize;
  const totalPages = Math.ceil(totalItems / effectivePageSize);
  const isAllSelected = pageSize === 'All';

  const startItem = isAllSelected ? 1 : (currentPage - 1) * effectivePageSize + 1;
  const endItem = isAllSelected ? totalItems : Math.min(currentPage * effectivePageSize, totalItems);

  return (
    <div className="flex items-center justify-between px-6 py-3 bg-gray-50 border-t border-gray-200">
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <label htmlFor="pageSize" className="text-sm text-gray-500">Rows per page:</label>
          <select
            id="pageSize"
            value={pageSize}
            onChange={(e) => onPageSizeChange(e.target.value === 'All' ? 'All' : Number(e.target.value) as PageSize)}
            className="text-sm border border-gray-300 rounded px-2 py-1 bg-white"
          >
            {PAGE_SIZE_OPTIONS.map((option) => (
              <option key={option} value={option}>{option}</option>
            ))}
          </select>
        </div>
        <div className="text-sm text-gray-500">
          Showing {startItem} to {endItem} of {totalItems} rows
        </div>
      </div>
      {!isAllSelected && totalPages > 1 && (
        <div className="flex items-center gap-2">
          <button
            onClick={() => onPageChange(1)}
            disabled={currentPage === 1}
            className="px-2 py-1 text-base text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50 disabled:cursor-not-allowed"
          >
            «
          </button>
          <button
            onClick={() => onPageChange(currentPage - 1)}
            disabled={currentPage === 1}
            className="px-2 py-1 text-base text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50 disabled:cursor-not-allowed"
          >
            ‹
          </button>
          <span className="px-3 py-1 text-sm text-gray-700">
            Page {currentPage} of {totalPages}
          </span>
          <button
            onClick={() => onPageChange(currentPage + 1)}
            disabled={currentPage === totalPages}
            className="px-2 py-1 text-base text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50 disabled:cursor-not-allowed"
          >
            ›
          </button>
          <button
            onClick={() => onPageChange(totalPages)}
            disabled={currentPage === totalPages}
            className="px-2 py-1 text-base text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50 disabled:cursor-not-allowed"
          >
            »
          </button>
        </div>
      )}
    </div>
  );
};

function paginate<T>(data: T[], page: number, pageSize: PageSize): T[] {
  if (pageSize === 'All') return data;
  const start = (page - 1) * pageSize;
  return data.slice(start, start + pageSize);
}

const SortIcon: React.FC<{ active: boolean; direction: SortDirection }> = ({ active, direction }) => {
  if (!active) return null;
  return (
    <span className="ml-1 inline-block text-blue-600">
      {direction === 'asc' ? '▲' : '▼'}
    </span>
  );
};

const SortableHeader: React.FC<{
  label: string;
  sortKey: string;
  currentSort: SortConfig;
  onSort: (key: string) => void;
  align?: 'left' | 'right';
  rowSpan?: number;
}> = ({ label, sortKey, currentSort, onSort, align = 'left', rowSpan }) => {
  const isActive = currentSort.key === sortKey;
  return (
    <th
      onClick={() => onSort(sortKey)}
      rowSpan={rowSpan}
      className={`px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none whitespace-nowrap ${
        align === 'right' ? 'text-right' : 'text-left'
      }`}
    >
      {label}
      <SortIcon active={isActive} direction={isActive ? currentSort.direction : 'asc'} />
    </th>
  );
};

const CostGroupHeader: React.FC<{
  sortKey: string;
  currentSort: SortConfig;
  onSort: (key: string) => void;
}> = ({ sortKey, currentSort, onSort }) => {
  const isActive = currentSort.key === sortKey;
  return (
    <th
      colSpan={3}
      onClick={() => onSort(sortKey)}
      className="px-6 py-2 text-center text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none whitespace-nowrap border-b border-gray-200"
    >
      Cost
      <SortIcon active={isActive} direction={isActive ? currentSort.direction : 'asc'} />
    </th>
  );
};

const CostSubHeaders: React.FC = () => (
  <>
    <th className="px-6 py-2 text-right text-xs font-medium text-gray-400 tracking-wider">Hourly</th>
    <th className="px-6 py-2 text-right text-xs font-medium text-gray-400 tracking-wider">Daily</th>
    <th className="px-6 py-2 text-right text-xs font-medium text-gray-400 tracking-wider">Monthly</th>
  </>
);

function sortData<T>(data: T[], sortConfig: SortConfig): T[] {
  return [...data].sort((a, b) => {
    const aVal = (a as Record<string, unknown>)[sortConfig.key];
    const bVal = (b as Record<string, unknown>)[sortConfig.key];

    let comparison = 0;
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      comparison = aVal.localeCompare(bVal);
    } else if (typeof aVal === 'number' && typeof bVal === 'number') {
      comparison = aVal - bVal;
    }

    return sortConfig.direction === 'asc' ? comparison : -comparison;
  });
}

export const CostTable: React.FC<CostTableProps> = ({
  accounts,
  regions,
  ec2,
  ebs,
  ecs,
  rds,
  eks,
  elb,
  nat,
  eip,
  secrets,
}) => {
  const [accountSort, setAccountSort] = useState<SortConfig>({ key: 'accountName', direction: 'asc' });
  const [regionSort, setRegionSort] = useState<SortConfig>({ key: 'region', direction: 'asc' });
  const [ec2Sort, setEc2Sort] = useState<SortConfig>({ key: 'name', direction: 'asc' });
  const [ebsSort, setEbsSort] = useState<SortConfig>({ key: 'name', direction: 'asc' });
  const [ecsSort, setEcsSort] = useState<SortConfig>({ key: 'serviceName', direction: 'asc' });
  const [rdsSort, setRdsSort] = useState<SortConfig>({ key: 'name', direction: 'asc' });
  const [eksSort, setEksSort] = useState<SortConfig>({ key: 'clusterName', direction: 'asc' });
  const [elbSort, setElbSort] = useState<SortConfig>({ key: 'name', direction: 'asc' });
  const [natSort, setNatSort] = useState<SortConfig>({ key: 'name', direction: 'asc' });
  const [eipSort, setEipSort] = useState<SortConfig>({ key: 'publicIp', direction: 'asc' });
  const [secretsSort, setSecretsSort] = useState<SortConfig>({ key: 'name', direction: 'asc' });

  const [accountPage, setAccountPage] = useState(1);
  const [regionPage, setRegionPage] = useState(1);
  const [ec2Page, setEc2Page] = useState(1);
  const [ebsPage, setEbsPage] = useState(1);
  const [ecsPage, setEcsPage] = useState(1);
  const [rdsPage, setRdsPage] = useState(1);
  const [eksPage, setEksPage] = useState(1);
  const [elbPage, setElbPage] = useState(1);
  const [natPage, setNatPage] = useState(1);
  const [eipPage, setEipPage] = useState(1);
  const [secretsPage, setSecretsPage] = useState(1);

  const [pageSize, setPageSize] = useState<PageSize>(10);

  const handlePageSizeChange = (newSize: PageSize, resetPage: () => void) => {
    setPageSize(newSize);
    resetPage();
  };

  const handleSort = (
    setter: React.Dispatch<React.SetStateAction<SortConfig>>,
    current: SortConfig,
    key: string,
    resetPage: () => void
  ) => {
    setter({
      key,
      direction: current.key === key && current.direction === 'asc' ? 'desc' : 'asc',
    });
    resetPage();
  };

  const formatCost = (cost: number, decimals: number = 4) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    }).format(cost);
  };

  const dailyCost = (hourly: number) => hourly * 24;
  const monthlyCost = (hourly: number) => hourly * 730;

  const sortedAccounts = useMemo(() => {
    if (!accounts) return [];
    return sortData(accounts, accountSort);
  }, [accounts, accountSort]);

  const sortedRegions = useMemo(() => {
    if (!regions) return [];
    return sortData(regions, regionSort);
  }, [regions, regionSort]);

  const sortedEc2 = useMemo(() => {
    if (!ec2) return [];
    return sortData(ec2, ec2Sort);
  }, [ec2, ec2Sort]);

  const sortedEbs = useMemo(() => {
    if (!ebs) return [];
    return sortData(ebs, ebsSort);
  }, [ebs, ebsSort]);

  const sortedEcs = useMemo(() => {
    if (!ecs) return [];
    return sortData(ecs, ecsSort);
  }, [ecs, ecsSort]);

  const sortedRds = useMemo(() => {
    if (!rds) return [];
    return sortData(rds, rdsSort);
  }, [rds, rdsSort]);

  const sortedEks = useMemo(() => {
    if (!eks) return [];
    return sortData(eks, eksSort);
  }, [eks, eksSort]);

  const sortedElb = useMemo(() => {
    if (!elb) return [];
    return sortData(elb, elbSort);
  }, [elb, elbSort]);

  const sortedNat = useMemo(() => {
    if (!nat) return [];
    return sortData(nat, natSort);
  }, [nat, natSort]);

  const sortedEip = useMemo(() => {
    if (!eip) return [];
    return sortData(eip, eipSort);
  }, [eip, eipSort]);

  const sortedSecrets = useMemo(() => {
    if (!secrets) return [];
    return sortData(secrets, secretsSort);
  }, [secrets, secretsSort]);

  // Accounts table
  if (accounts && accounts.length > 0) {
    const paginatedAccounts = paginate(sortedAccounts, accountPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={accountSort} onSort={(k) => handleSort(setAccountSort, accountSort, k, () => setAccountPage(1))} rowSpan={2} />
              <SortableHeader label="EC2" sortKey="ec2Count" currentSort={accountSort} onSort={(k) => handleSort(setAccountSort, accountSort, k, () => setAccountPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="EBS" sortKey="ebsCount" currentSort={accountSort} onSort={(k) => handleSort(setAccountSort, accountSort, k, () => setAccountPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="ECS" sortKey="ecsCount" currentSort={accountSort} onSort={(k) => handleSort(setAccountSort, accountSort, k, () => setAccountPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="RDS" sortKey="rdsCount" currentSort={accountSort} onSort={(k) => handleSort(setAccountSort, accountSort, k, () => setAccountPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="EKS" sortKey="eksCount" currentSort={accountSort} onSort={(k) => handleSort(setAccountSort, accountSort, k, () => setAccountPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="ELB" sortKey="elbCount" currentSort={accountSort} onSort={(k) => handleSort(setAccountSort, accountSort, k, () => setAccountPage(1))} rowSpan={2} align="right" />
              <CostGroupHeader sortKey="totalCost" currentSort={accountSort} onSort={(k) => handleSort(setAccountSort, accountSort, k, () => setAccountPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedAccounts.map((account) => (
              <tr key={account.accountId}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{account.accountName || account.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{account.ec2Count}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{account.ebsCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{account.ecsCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{account.rdsCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{account.eksCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{account.elbCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(account.totalCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(account.totalCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(account.totalCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={accountPage} totalItems={sortedAccounts.length} pageSize={pageSize} onPageChange={setAccountPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setAccountPage(1))} />
      </div>
    );
  }

  // Regions table
  if (regions && regions.length > 0) {
    const paginatedRegions = paginate(sortedRegions, regionPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Region" sortKey="region" currentSort={regionSort} onSort={(k) => handleSort(setRegionSort, regionSort, k, () => setRegionPage(1))} rowSpan={2} />
              <SortableHeader label="EC2" sortKey="ec2Count" currentSort={regionSort} onSort={(k) => handleSort(setRegionSort, regionSort, k, () => setRegionPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="EBS" sortKey="ebsCount" currentSort={regionSort} onSort={(k) => handleSort(setRegionSort, regionSort, k, () => setRegionPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="ECS" sortKey="ecsCount" currentSort={regionSort} onSort={(k) => handleSort(setRegionSort, regionSort, k, () => setRegionPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="RDS" sortKey="rdsCount" currentSort={regionSort} onSort={(k) => handleSort(setRegionSort, regionSort, k, () => setRegionPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="EKS" sortKey="eksCount" currentSort={regionSort} onSort={(k) => handleSort(setRegionSort, regionSort, k, () => setRegionPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="ELB" sortKey="elbCount" currentSort={regionSort} onSort={(k) => handleSort(setRegionSort, regionSort, k, () => setRegionPage(1))} rowSpan={2} align="right" />
              <CostGroupHeader sortKey="totalCost" currentSort={regionSort} onSort={(k) => handleSort(setRegionSort, regionSort, k, () => setRegionPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedRegions.map((region) => (
              <tr key={region.region}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{region.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{region.ec2Count}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{region.ebsCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{region.ecsCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{region.rdsCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{region.eksCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{region.elbCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(region.totalCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(region.totalCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(region.totalCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={regionPage} totalItems={sortedRegions.length} pageSize={pageSize} onPageChange={setRegionPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setRegionPage(1))} />
      </div>
    );
  }

  // EC2 table
  if (ec2 && ec2.length > 0) {
    const paginatedEc2 = paginate(sortedEc2, ec2Page, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={ec2Sort} onSort={(k) => handleSort(setEc2Sort, ec2Sort, k, () => setEc2Page(1))} rowSpan={2} />
              <SortableHeader label="Region" sortKey="region" currentSort={ec2Sort} onSort={(k) => handleSort(setEc2Sort, ec2Sort, k, () => setEc2Page(1))} rowSpan={2} />
              <SortableHeader label="Name" sortKey="name" currentSort={ec2Sort} onSort={(k) => handleSort(setEc2Sort, ec2Sort, k, () => setEc2Page(1))} rowSpan={2} />
              <SortableHeader label="Instance ID" sortKey="instanceId" currentSort={ec2Sort} onSort={(k) => handleSort(setEc2Sort, ec2Sort, k, () => setEc2Page(1))} rowSpan={2} />
              <SortableHeader label="Type" sortKey="instanceType" currentSort={ec2Sort} onSort={(k) => handleSort(setEc2Sort, ec2Sort, k, () => setEc2Page(1))} rowSpan={2} />
              <SortableHeader label="State" sortKey="state" currentSort={ec2Sort} onSort={(k) => handleSort(setEc2Sort, ec2Sort, k, () => setEc2Page(1))} rowSpan={2} />
              <CostGroupHeader sortKey="hourlyCost" currentSort={ec2Sort} onSort={(k) => handleSort(setEc2Sort, ec2Sort, k, () => setEc2Page(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedEc2.map((inst) => (
              <tr key={inst.instanceId}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inst.accountName || inst.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inst.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{inst.name || '-'}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inst.instanceId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inst.instanceType}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    inst.state === 'running' ? 'bg-green-100 text-green-800' :
                    inst.state === 'stopped' ? 'bg-red-100 text-red-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {inst.state}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(inst.hourlyCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(inst.hourlyCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(inst.hourlyCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={ec2Page} totalItems={sortedEc2.length} pageSize={pageSize} onPageChange={setEc2Page} onPageSizeChange={(size) => handlePageSizeChange(size, () => setEc2Page(1))} />
      </div>
    );
  }

  // EBS table
  if (ebs && ebs.length > 0) {
    const paginatedEbs = paginate(sortedEbs, ebsPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={ebsSort} onSort={(k) => handleSort(setEbsSort, ebsSort, k, () => setEbsPage(1))} rowSpan={2} />
              <SortableHeader label="Region" sortKey="region" currentSort={ebsSort} onSort={(k) => handleSort(setEbsSort, ebsSort, k, () => setEbsPage(1))} rowSpan={2} />
              <SortableHeader label="Name" sortKey="name" currentSort={ebsSort} onSort={(k) => handleSort(setEbsSort, ebsSort, k, () => setEbsPage(1))} rowSpan={2} />
              <SortableHeader label="Volume ID" sortKey="volumeId" currentSort={ebsSort} onSort={(k) => handleSort(setEbsSort, ebsSort, k, () => setEbsPage(1))} rowSpan={2} />
              <SortableHeader label="Type" sortKey="volumeType" currentSort={ebsSort} onSort={(k) => handleSort(setEbsSort, ebsSort, k, () => setEbsPage(1))} rowSpan={2} />
              <SortableHeader label="Size (GiB)" sortKey="size" currentSort={ebsSort} onSort={(k) => handleSort(setEbsSort, ebsSort, k, () => setEbsPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="State" sortKey="state" currentSort={ebsSort} onSort={(k) => handleSort(setEbsSort, ebsSort, k, () => setEbsPage(1))} rowSpan={2} />
              <CostGroupHeader sortKey="hourlyCost" currentSort={ebsSort} onSort={(k) => handleSort(setEbsSort, ebsSort, k, () => setEbsPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedEbs.map((vol) => (
              <tr key={vol.volumeId}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{vol.accountName || vol.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{vol.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{vol.name || '-'}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{vol.volumeId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{vol.volumeType}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{vol.size}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    vol.state === 'in-use' ? 'bg-green-100 text-green-800' :
                    vol.state === 'available' ? 'bg-blue-100 text-blue-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {vol.state}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(vol.hourlyCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(vol.hourlyCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(vol.hourlyCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={ebsPage} totalItems={sortedEbs.length} pageSize={pageSize} onPageChange={setEbsPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setEbsPage(1))} />
      </div>
    );
  }

  // ECS table
  if (ecs && ecs.length > 0) {
    const paginatedEcs = paginate(sortedEcs, ecsPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={ecsSort} onSort={(k) => handleSort(setEcsSort, ecsSort, k, () => setEcsPage(1))} rowSpan={2} />
              <SortableHeader label="Region" sortKey="region" currentSort={ecsSort} onSort={(k) => handleSort(setEcsSort, ecsSort, k, () => setEcsPage(1))} rowSpan={2} />
              <SortableHeader label="Cluster" sortKey="clusterName" currentSort={ecsSort} onSort={(k) => handleSort(setEcsSort, ecsSort, k, () => setEcsPage(1))} rowSpan={2} />
              <SortableHeader label="Service" sortKey="serviceName" currentSort={ecsSort} onSort={(k) => handleSort(setEcsSort, ecsSort, k, () => setEcsPage(1))} rowSpan={2} />
              <SortableHeader label="Launch Type" sortKey="launchType" currentSort={ecsSort} onSort={(k) => handleSort(setEcsSort, ecsSort, k, () => setEcsPage(1))} rowSpan={2} />
              <SortableHeader label="Desired" sortKey="desiredCount" currentSort={ecsSort} onSort={(k) => handleSort(setEcsSort, ecsSort, k, () => setEcsPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="Running" sortKey="runningCount" currentSort={ecsSort} onSort={(k) => handleSort(setEcsSort, ecsSort, k, () => setEcsPage(1))} rowSpan={2} align="right" />
              <SortableHeader label="State" sortKey="state" currentSort={ecsSort} onSort={(k) => handleSort(setEcsSort, ecsSort, k, () => setEcsPage(1))} rowSpan={2} />
              <CostGroupHeader sortKey="hourlyCost" currentSort={ecsSort} onSort={(k) => handleSort(setEcsSort, ecsSort, k, () => setEcsPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedEcs.map((svc) => (
              <tr key={`${svc.clusterName}-${svc.serviceName}`}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{svc.accountName || svc.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{svc.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{svc.clusterName}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{svc.serviceName}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{svc.launchType}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{svc.desiredCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{svc.runningCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    svc.state === 'ACTIVE' ? 'bg-green-100 text-green-800' :
                    svc.state === 'DRAINING' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {svc.state}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(svc.hourlyCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(svc.hourlyCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(svc.hourlyCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={ecsPage} totalItems={sortedEcs.length} pageSize={pageSize} onPageChange={setEcsPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setEcsPage(1))} />
      </div>
    );
  }

  // RDS table
  if (rds && rds.length > 0) {
    const paginatedRds = paginate(sortedRds, rdsPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={rdsSort} onSort={(k) => handleSort(setRdsSort, rdsSort, k, () => setRdsPage(1))} rowSpan={2} />
              <SortableHeader label="Region" sortKey="region" currentSort={rdsSort} onSort={(k) => handleSort(setRdsSort, rdsSort, k, () => setRdsPage(1))} rowSpan={2} />
              <SortableHeader label="Name" sortKey="name" currentSort={rdsSort} onSort={(k) => handleSort(setRdsSort, rdsSort, k, () => setRdsPage(1))} rowSpan={2} />
              <SortableHeader label="Engine" sortKey="engine" currentSort={rdsSort} onSort={(k) => handleSort(setRdsSort, rdsSort, k, () => setRdsPage(1))} rowSpan={2} />
              <SortableHeader label="Class" sortKey="instanceClass" currentSort={rdsSort} onSort={(k) => handleSort(setRdsSort, rdsSort, k, () => setRdsPage(1))} rowSpan={2} />
              <SortableHeader label="Multi-AZ" sortKey="multiAz" currentSort={rdsSort} onSort={(k) => handleSort(setRdsSort, rdsSort, k, () => setRdsPage(1))} rowSpan={2} />
              <SortableHeader label="State" sortKey="state" currentSort={rdsSort} onSort={(k) => handleSort(setRdsSort, rdsSort, k, () => setRdsPage(1))} rowSpan={2} />
              <CostGroupHeader sortKey="hourlyCost" currentSort={rdsSort} onSort={(k) => handleSort(setRdsSort, rdsSort, k, () => setRdsPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedRds.map((inst) => (
              <tr key={inst.dbInstanceId}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inst.accountName || inst.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inst.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{inst.name}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inst.engine}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inst.instanceClass}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inst.multiAz ? 'Yes' : 'No'}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    inst.state === 'available' ? 'bg-green-100 text-green-800' :
                    inst.state === 'stopped' ? 'bg-red-100 text-red-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {inst.state}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(inst.hourlyCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(inst.hourlyCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(inst.hourlyCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={rdsPage} totalItems={sortedRds.length} pageSize={pageSize} onPageChange={setRdsPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setRdsPage(1))} />
      </div>
    );
  }

  // EKS table
  if (eks && eks.length > 0) {
    const paginatedEks = paginate(sortedEks, eksPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={eksSort} onSort={(k) => handleSort(setEksSort, eksSort, k, () => setEksPage(1))} rowSpan={2} />
              <SortableHeader label="Region" sortKey="region" currentSort={eksSort} onSort={(k) => handleSort(setEksSort, eksSort, k, () => setEksPage(1))} rowSpan={2} />
              <SortableHeader label="Cluster" sortKey="clusterName" currentSort={eksSort} onSort={(k) => handleSort(setEksSort, eksSort, k, () => setEksPage(1))} rowSpan={2} />
              <SortableHeader label="Status" sortKey="status" currentSort={eksSort} onSort={(k) => handleSort(setEksSort, eksSort, k, () => setEksPage(1))} rowSpan={2} />
              <SortableHeader label="Version" sortKey="version" currentSort={eksSort} onSort={(k) => handleSort(setEksSort, eksSort, k, () => setEksPage(1))} rowSpan={2} />
              <SortableHeader label="Platform" sortKey="platform" currentSort={eksSort} onSort={(k) => handleSort(setEksSort, eksSort, k, () => setEksPage(1))} rowSpan={2} />
              <CostGroupHeader sortKey="hourlyCost" currentSort={eksSort} onSort={(k) => handleSort(setEksSort, eksSort, k, () => setEksPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedEks.map((cluster) => (
              <tr key={`${cluster.accountId}-${cluster.region}-${cluster.clusterName}`}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{cluster.accountName || cluster.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{cluster.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{cluster.clusterName}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    cluster.status === 'ACTIVE' ? 'bg-green-100 text-green-800' :
                    cluster.status === 'CREATING' ? 'bg-yellow-100 text-yellow-800' :
                    cluster.status === 'DELETING' ? 'bg-red-100 text-red-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {cluster.status}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{cluster.version}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{cluster.platform}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(cluster.hourlyCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(cluster.hourlyCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(cluster.hourlyCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={eksPage} totalItems={sortedEks.length} pageSize={pageSize} onPageChange={setEksPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setEksPage(1))} />
      </div>
    );
  }

  // ELB table
  if (elb && elb.length > 0) {
    const paginatedElb = paginate(sortedElb, elbPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={elbSort} onSort={(k) => handleSort(setElbSort, elbSort, k, () => setElbPage(1))} rowSpan={2} />
              <SortableHeader label="Region" sortKey="region" currentSort={elbSort} onSort={(k) => handleSort(setElbSort, elbSort, k, () => setElbPage(1))} rowSpan={2} />
              <SortableHeader label="Name" sortKey="name" currentSort={elbSort} onSort={(k) => handleSort(setElbSort, elbSort, k, () => setElbPage(1))} rowSpan={2} />
              <SortableHeader label="Type" sortKey="type" currentSort={elbSort} onSort={(k) => handleSort(setElbSort, elbSort, k, () => setElbPage(1))} rowSpan={2} />
              <SortableHeader label="Scheme" sortKey="scheme" currentSort={elbSort} onSort={(k) => handleSort(setElbSort, elbSort, k, () => setElbPage(1))} rowSpan={2} />
              <SortableHeader label="State" sortKey="state" currentSort={elbSort} onSort={(k) => handleSort(setElbSort, elbSort, k, () => setElbPage(1))} rowSpan={2} />
              <CostGroupHeader sortKey="hourlyCost" currentSort={elbSort} onSort={(k) => handleSort(setElbSort, elbSort, k, () => setElbPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedElb.map((lb) => (
              <tr key={lb.arn || `${lb.accountId}-${lb.region}-${lb.name}`}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{lb.accountName || lb.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{lb.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{lb.name}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    lb.type === 'application' ? 'bg-blue-100 text-blue-800' :
                    lb.type === 'network' ? 'bg-purple-100 text-purple-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {lb.type}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{lb.scheme}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    lb.state === 'active' ? 'bg-green-100 text-green-800' :
                    lb.state === 'provisioning' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {lb.state}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(lb.hourlyCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(lb.hourlyCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(lb.hourlyCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={elbPage} totalItems={sortedElb.length} pageSize={pageSize} onPageChange={setElbPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setElbPage(1))} />
      </div>
    );
  }

  // NAT Gateway table
  if (nat && nat.length > 0) {
    const paginatedNat = paginate(sortedNat, natPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={natSort} onSort={(k) => handleSort(setNatSort, natSort, k, () => setNatPage(1))} rowSpan={2} />
              <SortableHeader label="Region" sortKey="region" currentSort={natSort} onSort={(k) => handleSort(setNatSort, natSort, k, () => setNatPage(1))} rowSpan={2} />
              <SortableHeader label="Name" sortKey="name" currentSort={natSort} onSort={(k) => handleSort(setNatSort, natSort, k, () => setNatPage(1))} rowSpan={2} />
              <SortableHeader label="ID" sortKey="id" currentSort={natSort} onSort={(k) => handleSort(setNatSort, natSort, k, () => setNatPage(1))} rowSpan={2} />
              <SortableHeader label="State" sortKey="state" currentSort={natSort} onSort={(k) => handleSort(setNatSort, natSort, k, () => setNatPage(1))} rowSpan={2} />
              <SortableHeader label="Type" sortKey="type" currentSort={natSort} onSort={(k) => handleSort(setNatSort, natSort, k, () => setNatPage(1))} rowSpan={2} />
              <SortableHeader label="VPC ID" sortKey="vpcId" currentSort={natSort} onSort={(k) => handleSort(setNatSort, natSort, k, () => setNatPage(1))} rowSpan={2} />
              <CostGroupHeader sortKey="hourlyCost" currentSort={natSort} onSort={(k) => handleSort(setNatSort, natSort, k, () => setNatPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedNat.map((gateway) => (
              <tr key={gateway.id}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{gateway.accountName || gateway.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{gateway.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{gateway.name || '-'}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{gateway.id}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    gateway.state === 'available' ? 'bg-green-100 text-green-800' :
                    gateway.state === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {gateway.state}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{gateway.type}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{gateway.vpcId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(gateway.hourlyCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(gateway.hourlyCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(gateway.hourlyCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={natPage} totalItems={sortedNat.length} pageSize={pageSize} onPageChange={setNatPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setNatPage(1))} />
      </div>
    );
  }

  // Elastic IP table
  if (eip && eip.length > 0) {
    const paginatedEip = paginate(sortedEip, eipPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={eipSort} onSort={(k) => handleSort(setEipSort, eipSort, k, () => setEipPage(1))} rowSpan={2} />
              <SortableHeader label="Region" sortKey="region" currentSort={eipSort} onSort={(k) => handleSort(setEipSort, eipSort, k, () => setEipPage(1))} rowSpan={2} />
              <SortableHeader label="Name" sortKey="name" currentSort={eipSort} onSort={(k) => handleSort(setEipSort, eipSort, k, () => setEipPage(1))} rowSpan={2} />
              <SortableHeader label="Public IP" sortKey="publicIp" currentSort={eipSort} onSort={(k) => handleSort(setEipSort, eipSort, k, () => setEipPage(1))} rowSpan={2} />
              <SortableHeader label="Allocation ID" sortKey="allocationId" currentSort={eipSort} onSort={(k) => handleSort(setEipSort, eipSort, k, () => setEipPage(1))} rowSpan={2} />
              <SortableHeader label="Associated" sortKey="isAssociated" currentSort={eipSort} onSort={(k) => handleSort(setEipSort, eipSort, k, () => setEipPage(1))} rowSpan={2} />
              <SortableHeader label="Instance ID" sortKey="instanceId" currentSort={eipSort} onSort={(k) => handleSort(setEipSort, eipSort, k, () => setEipPage(1))} rowSpan={2} />
              <CostGroupHeader sortKey="hourlyCost" currentSort={eipSort} onSort={(k) => handleSort(setEipSort, eipSort, k, () => setEipPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedEip.map((ip) => (
              <tr key={ip.allocationId}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{ip.accountName || ip.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{ip.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{ip.name || '-'}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{ip.publicIp}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{ip.allocationId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    ip.isAssociated ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                  }`}>
                    {ip.isAssociated ? 'Yes' : 'No'}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{ip.instanceId || '-'}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(ip.hourlyCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(ip.hourlyCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(ip.hourlyCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={eipPage} totalItems={sortedEip.length} pageSize={pageSize} onPageChange={setEipPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setEipPage(1))} />
      </div>
    );
  }

  // Secrets table
  if (secrets && secrets.length > 0) {
    const paginatedSecrets = paginate(sortedSecrets, secretsPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader label="Account" sortKey="accountName" currentSort={secretsSort} onSort={(k) => handleSort(setSecretsSort, secretsSort, k, () => setSecretsPage(1))} rowSpan={2} />
              <SortableHeader label="Region" sortKey="region" currentSort={secretsSort} onSort={(k) => handleSort(setSecretsSort, secretsSort, k, () => setSecretsPage(1))} rowSpan={2} />
              <SortableHeader label="Name" sortKey="name" currentSort={secretsSort} onSort={(k) => handleSort(setSecretsSort, secretsSort, k, () => setSecretsPage(1))} rowSpan={2} />
              <SortableHeader label="Description" sortKey="description" currentSort={secretsSort} onSort={(k) => handleSort(setSecretsSort, secretsSort, k, () => setSecretsPage(1))} rowSpan={2} />
              <CostGroupHeader sortKey="hourlyCost" currentSort={secretsSort} onSort={(k) => handleSort(setSecretsSort, secretsSort, k, () => setSecretsPage(1))} />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedSecrets.map((secret) => (
              <tr key={secret.arn}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{secret.accountName || secret.accountId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{secret.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{secret.name}</td>
                <td className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate">{secret.description || '-'}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(secret.hourlyCost)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(dailyCost(secret.hourlyCost), 2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">{formatCost(monthlyCost(secret.hourlyCost), 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination currentPage={secretsPage} totalItems={sortedSecrets.length} pageSize={pageSize} onPageChange={setSecretsPage} onPageSizeChange={(size) => handlePageSizeChange(size, () => setSecretsPage(1))} />
      </div>
    );
  }

  return (
    <div className="p-6">
      <p className="text-gray-500 text-center">No data available</p>
    </div>
  );
};

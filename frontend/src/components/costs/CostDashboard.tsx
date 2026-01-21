import React, { useEffect, useState, useMemo, useRef } from 'react';
import { useAppSelector, useAppDispatch } from '../../hooks/useAppDispatch';
import { fetchCosts, fetchConfig } from '../../store/costSlice';
import { CostSummary } from './CostSummary';
import { CostTable } from './CostTable';
import { ResourceSelector } from './ResourceSelector';

type TabType = 'accounts' | 'regions' | 'ec2' | 'ebs' | 'ecs' | 'rds' | 'eks' | 'elb' | 'nat' | 'eip' | 'secrets' | 'publicipv4';

const allTabs: { id: TabType; label: string }[] = [
  { id: 'accounts', label: 'Accounts' },
  { id: 'regions', label: 'Regions' },
  { id: 'ec2', label: 'EC2' },
  { id: 'ebs', label: 'EBS' },
  { id: 'ecs', label: 'ECS' },
  { id: 'rds', label: 'RDS' },
  { id: 'eks', label: 'EKS' },
  { id: 'elb', label: 'ELB' },
  { id: 'nat', label: 'NAT' },
  { id: 'eip', label: 'EIP' },
  { id: 'secrets', label: 'Secrets' },
  { id: 'publicipv4', label: 'Public IPv4' },
];

export const CostDashboard: React.FC = () => {
  const dispatch = useAppDispatch();
  const { data, loading, error, hasLoadedData, selectedAccounts, selectedRegions, selectedResources } = useAppSelector((state) => state.costs);
  const [activeTab, setActiveTab] = useState<TabType>('accounts');
  const [filter, setFilter] = useState('');
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  // Filter tabs based on selected resources (always show accounts and regions)
  const tabs = useMemo(() => {
    return allTabs.filter((tab) => {
      if (tab.id === 'accounts' || tab.id === 'regions') return true;
      return selectedResources.includes(tab.id);
    });
  }, [selectedResources]);

  // Reset to accounts tab if current tab is no longer visible
  useEffect(() => {
    const tabIds = tabs.map((t) => t.id);
    if (!tabIds.includes(activeTab)) {
      setActiveTab('accounts');
    }
  }, [tabs, activeTab]);

  // Load config on mount
  useEffect(() => {
    dispatch(fetchConfig());
  }, [dispatch]);

  // Set up auto-refresh when data has been loaded
  useEffect(() => {
    if (hasLoadedData && selectedAccounts.length > 0 && selectedRegions.length > 0 && selectedResources.length > 0) {
      // Clear any existing interval
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
      // Auto-refresh every 5 minutes
      intervalRef.current = setInterval(() => {
        dispatch(fetchCosts());
      }, 5 * 60 * 1000);
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [dispatch, hasLoadedData, selectedAccounts, selectedRegions, selectedResources]);

  // Filter data based on active tab and filter text
  const filteredData = useMemo(() => {
    if (!data) return null;

    const matchesFilter = (searchableFields: string[]): boolean => {
      if (!filter.trim()) return true;

      const terms = filter.toLowerCase().split(/\s+/).filter(t => t);
      const combined = searchableFields.join(' ').toLowerCase();

      for (const term of terms) {
        if (term.startsWith('!') && term.length > 1) {
          const searchTerm = term.slice(1);
          if (combined.includes(searchTerm)) return false;
        } else {
          if (!combined.includes(term)) return false;
        }
      }
      return true;
    };

    return {
      accounts: data.accounts?.filter((a) =>
        matchesFilter([a.accountName, a.accountId])
      ),
      regions: data.regions?.filter((r) =>
        matchesFilter([r.region])
      ),
      ec2: data.ec2Instances?.filter((inst) =>
        matchesFilter([inst.name, inst.instanceId, inst.instanceType, inst.region, inst.accountName])
      ),
      ebs: data.ebsVolumes?.filter((vol) =>
        matchesFilter([vol.name, vol.volumeId, vol.volumeType, vol.region, vol.accountName])
      ),
      ecs: data.ecsServices?.filter((svc) =>
        matchesFilter([svc.serviceName, svc.clusterName, svc.launchType, svc.region, svc.accountName])
      ),
      rds: data.rdsInstances?.filter((inst) =>
        matchesFilter([inst.name, inst.dbInstanceId, inst.engine, inst.instanceClass, inst.region, inst.accountName])
      ),
      eks: data.eksClusters?.filter((cluster) =>
        matchesFilter([cluster.clusterName, cluster.version, cluster.status, cluster.region, cluster.accountName])
      ),
      elb: data.loadBalancers?.filter((lb) =>
        matchesFilter([lb.name, lb.type, lb.scheme, lb.state, lb.region, lb.accountName])
      ),
      nat: data.natGateways?.filter((nat) =>
        matchesFilter([nat.name, nat.id, nat.state, nat.type, nat.vpcId, nat.region, nat.accountName])
      ),
      eip: data.elasticIps?.filter((eip) =>
        matchesFilter([eip.name, eip.publicIp, eip.allocationId, eip.instanceId, eip.region, eip.accountName])
      ),
      secrets: data.secrets?.filter((secret) =>
        matchesFilter([secret.name, secret.description, secret.region, secret.accountName])
      ),
      publicipv4: data.publicIpv4s?.filter((pip) =>
        matchesFilter([pip.publicIp, pip.instanceId, pip.instanceName, pip.region, pip.accountName])
      ),
    };
  }, [data, filter]);

  const getTabCount = (tab: TabType): { filtered: number; total: number } => {
    if (!data) return { filtered: 0, total: 0 };
    switch (tab) {
      case 'accounts': return { filtered: filteredData?.accounts?.length || 0, total: data.accounts?.length || 0 };
      case 'regions': return { filtered: filteredData?.regions?.length || 0, total: data.regions?.length || 0 };
      case 'ec2': return { filtered: filteredData?.ec2?.length || 0, total: data.ec2Instances?.length || 0 };
      case 'ebs': return { filtered: filteredData?.ebs?.length || 0, total: data.ebsVolumes?.length || 0 };
      case 'ecs': return { filtered: filteredData?.ecs?.length || 0, total: data.ecsServices?.length || 0 };
      case 'rds': return { filtered: filteredData?.rds?.length || 0, total: data.rdsInstances?.length || 0 };
      case 'eks': return { filtered: filteredData?.eks?.length || 0, total: data.eksClusters?.length || 0 };
      case 'elb': return { filtered: filteredData?.elb?.length || 0, total: data.loadBalancers?.length || 0 };
      case 'nat': return { filtered: filteredData?.nat?.length || 0, total: data.natGateways?.length || 0 };
      case 'eip': return { filtered: filteredData?.eip?.length || 0, total: data.elasticIps?.length || 0 };
      case 'secrets': return { filtered: filteredData?.secrets?.length || 0, total: data.secrets?.length || 0 };
      case 'publicipv4': return { filtered: filteredData?.publicipv4?.length || 0, total: data.publicIpv4s?.length || 0 };
    }
  };

  const formatTabCount = (tab: TabType): string => {
    const { filtered, total } = getTabCount(tab);
    if (filtered === total) return String(total);
    return `${filtered}/${total}`;
  };

  // Calculate totals from all data (unfiltered)
  const totals = useMemo(() => {
    if (!data) return { cost: 0, count: 0 };
    const cost = data.totalCost;
    const count = (data.ec2Instances?.length || 0) + (data.ebsVolumes?.length || 0) +
      (data.ecsServices?.length || 0) + (data.rdsInstances?.length || 0) + (data.eksClusters?.length || 0) +
      (data.loadBalancers?.length || 0) + (data.natGateways?.length || 0) + (data.elasticIps?.length || 0) +
      (data.secrets?.length || 0) + (data.publicIpv4s?.length || 0);
    return { cost, count };
  }, [data]);

  // Calculate selected summary based on active tab and filter
  const selectedData = useMemo(() => {
    if (!filteredData) return { cost: 0, count: 0 };

    const sumCost = (items: { hourlyCost: number }[] | undefined) =>
      items?.reduce((sum, item) => sum + item.hourlyCost, 0) || 0;

    const isResourceTab = activeTab !== 'accounts' && activeTab !== 'regions';

    // For accounts/regions tabs, sum all filtered resource types
    if (!isResourceTab) {
      const cost = sumCost(filteredData.ec2) + sumCost(filteredData.ebs) + sumCost(filteredData.ecs) +
        sumCost(filteredData.rds) + sumCost(filteredData.eks) + sumCost(filteredData.elb) +
        sumCost(filteredData.nat) + sumCost(filteredData.eip) + sumCost(filteredData.secrets) +
        sumCost(filteredData.publicipv4);
      const count = (filteredData.ec2?.length || 0) + (filteredData.ebs?.length || 0) +
        (filteredData.ecs?.length || 0) + (filteredData.rds?.length || 0) + (filteredData.eks?.length || 0) +
        (filteredData.elb?.length || 0) + (filteredData.nat?.length || 0) + (filteredData.eip?.length || 0) +
        (filteredData.secrets?.length || 0) + (filteredData.publicipv4?.length || 0);
      return { cost, count };
    }

    // For specific resource tabs, show only that resource type's data
    let items: { hourlyCost: number }[] | undefined;
    switch (activeTab) {
      case 'ec2': items = filteredData.ec2; break;
      case 'ebs': items = filteredData.ebs; break;
      case 'ecs': items = filteredData.ecs; break;
      case 'rds': items = filteredData.rds; break;
      case 'eks': items = filteredData.eks; break;
      case 'elb': items = filteredData.elb; break;
      case 'nat': items = filteredData.nat; break;
      case 'eip': items = filteredData.eip; break;
      case 'secrets': items = filteredData.secrets; break;
      case 'publicipv4': items = filteredData.publicipv4; break;
    }

    return { cost: sumCost(items), count: items?.length || 0 };
  }, [filteredData, activeTab]);

  const exportToCSV = () => {
    if (!filteredData) return;

    let headers: string[] = [];
    let rows: string[][] = [];

    const dailyCost = (hourly: number) => hourly * 24;
    const monthlyCost = (hourly: number) => hourly * 730;

    switch (activeTab) {
      case 'accounts':
        headers = ['Account ID', 'Account Name', 'EC2', 'EBS', 'ECS', 'RDS', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.accounts || []).map(a => [
          a.accountId,
          a.accountName,
          String(a.ec2Count),
          String(a.ebsCount),
          String(a.ecsCount),
          String(a.rdsCount),
          a.totalCost.toFixed(4),
          dailyCost(a.totalCost).toFixed(2),
          monthlyCost(a.totalCost).toFixed(2),
        ]);
        break;
      case 'regions':
        headers = ['Region', 'EC2', 'EBS', 'ECS', 'RDS', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.regions || []).map(r => [
          r.region,
          String(r.ec2Count),
          String(r.ebsCount),
          String(r.ecsCount),
          String(r.rdsCount),
          r.totalCost.toFixed(4),
          dailyCost(r.totalCost).toFixed(2),
          monthlyCost(r.totalCost).toFixed(2),
        ]);
        break;
      case 'ec2':
        headers = ['Account', 'Region', 'Name', 'Instance ID', 'Type', 'State', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.ec2 || []).map(inst => [
          inst.accountName || inst.accountId,
          inst.region,
          inst.name,
          inst.instanceId,
          inst.instanceType,
          inst.state,
          inst.hourlyCost.toFixed(4),
          dailyCost(inst.hourlyCost).toFixed(2),
          monthlyCost(inst.hourlyCost).toFixed(2),
        ]);
        break;
      case 'ebs':
        headers = ['Account', 'Region', 'Name', 'Volume ID', 'Type', 'Size (GiB)', 'State', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.ebs || []).map(vol => [
          vol.accountName || vol.accountId,
          vol.region,
          vol.name,
          vol.volumeId,
          vol.volumeType,
          String(vol.size),
          vol.state,
          vol.hourlyCost.toFixed(4),
          dailyCost(vol.hourlyCost).toFixed(2),
          monthlyCost(vol.hourlyCost).toFixed(2),
        ]);
        break;
      case 'ecs':
        headers = ['Account', 'Region', 'Cluster', 'Service', 'Launch Type', 'Desired', 'Running', 'State', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.ecs || []).map(svc => [
          svc.accountName || svc.accountId,
          svc.region,
          svc.clusterName,
          svc.serviceName,
          svc.launchType,
          String(svc.desiredCount),
          String(svc.runningCount),
          svc.state,
          svc.hourlyCost.toFixed(4),
          dailyCost(svc.hourlyCost).toFixed(2),
          monthlyCost(svc.hourlyCost).toFixed(2),
        ]);
        break;
      case 'rds':
        headers = ['Account', 'Region', 'Name', 'Engine', 'Class', 'Multi-AZ', 'State', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.rds || []).map(inst => [
          inst.accountName || inst.accountId,
          inst.region,
          inst.name,
          inst.engine,
          inst.instanceClass,
          inst.multiAz ? 'Yes' : 'No',
          inst.state,
          inst.hourlyCost.toFixed(4),
          dailyCost(inst.hourlyCost).toFixed(2),
          monthlyCost(inst.hourlyCost).toFixed(2),
        ]);
        break;
      case 'eks':
        headers = ['Account', 'Region', 'Cluster', 'Status', 'Version', 'Platform', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.eks || []).map(cluster => [
          cluster.accountName || cluster.accountId,
          cluster.region,
          cluster.clusterName,
          cluster.status,
          cluster.version,
          cluster.platform,
          cluster.hourlyCost.toFixed(4),
          dailyCost(cluster.hourlyCost).toFixed(2),
          monthlyCost(cluster.hourlyCost).toFixed(2),
        ]);
        break;
      case 'elb':
        headers = ['Account', 'Region', 'Name', 'Type', 'Scheme', 'State', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.elb || []).map(lb => [
          lb.accountName || lb.accountId,
          lb.region,
          lb.name,
          lb.type,
          lb.scheme,
          lb.state,
          lb.hourlyCost.toFixed(4),
          dailyCost(lb.hourlyCost).toFixed(2),
          monthlyCost(lb.hourlyCost).toFixed(2),
        ]);
        break;
      case 'nat':
        headers = ['Account', 'Region', 'Name', 'ID', 'State', 'Type', 'VPC ID', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.nat || []).map(nat => [
          nat.accountName || nat.accountId,
          nat.region,
          nat.name,
          nat.id,
          nat.state,
          nat.type,
          nat.vpcId,
          nat.hourlyCost.toFixed(4),
          dailyCost(nat.hourlyCost).toFixed(2),
          monthlyCost(nat.hourlyCost).toFixed(2),
        ]);
        break;
      case 'eip':
        headers = ['Account', 'Region', 'Name', 'Public IP', 'Allocation ID', 'Associated', 'Instance ID', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.eip || []).map(eip => [
          eip.accountName || eip.accountId,
          eip.region,
          eip.name,
          eip.publicIp,
          eip.allocationId,
          eip.isAssociated ? 'Yes' : 'No',
          eip.instanceId || '',
          eip.hourlyCost.toFixed(4),
          dailyCost(eip.hourlyCost).toFixed(2),
          monthlyCost(eip.hourlyCost).toFixed(2),
        ]);
        break;
      case 'secrets':
        headers = ['Account', 'Region', 'Name', 'Description', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.secrets || []).map(secret => [
          secret.accountName || secret.accountId,
          secret.region,
          secret.name,
          secret.description || '',
          secret.hourlyCost.toFixed(4),
          dailyCost(secret.hourlyCost).toFixed(2),
          monthlyCost(secret.hourlyCost).toFixed(2),
        ]);
        break;
      case 'publicipv4':
        headers = ['Account', 'Region', 'Public IP', 'Instance ID', 'Instance Name', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.publicipv4 || []).map(pip => [
          pip.accountName || pip.accountId,
          pip.region,
          pip.publicIp,
          pip.instanceId || '',
          pip.instanceName || '',
          pip.hourlyCost.toFixed(4),
          dailyCost(pip.hourlyCost).toFixed(2),
          monthlyCost(pip.hourlyCost).toFixed(2),
        ]);
        break;
    }

    const escapeCSV = (value: string) => {
      if (value.includes(',') || value.includes('"') || value.includes('\n')) {
        return `"${value.replace(/"/g, '""')}"`;
      }
      return value;
    };

    const csvContent = [
      headers.map(escapeCSV).join(','),
      ...rows.map(row => row.map(escapeCSV).join(','))
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    link.setAttribute('download', `awscogs-${activeTab}-${new Date().toISOString().split('T')[0]}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <div>
      {/* Resource Selector */}
      <ResourceSelector />

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-6">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Error loading costs</h3>
              <p className="mt-1 text-sm text-red-700">{error}</p>
            </div>
          </div>
        </div>
      )}

      {/* Loading State - only show when loading and no previous data */}
      {loading && !data && hasLoadedData && (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      )}

      {/* Cost Data */}
      {data && (
        <>
          <CostSummary
            selectedCost={selectedData.cost}
            totalCost={totals.cost}
            selectedCount={selectedData.count}
            totalCount={totals.count}
            currency={data.currency}
          />

          {/* Tabs and Filter */}
          <div className="bg-white shadow rounded-lg">
            <div className="border-b border-gray-200">
              <div className="flex items-center px-4">
                {/* Tab Buttons - scrollable container */}
                <nav className="flex overflow-x-auto flex-1 min-w-0">
                  {tabs.map((tab) => (
                    <button
                      key={tab.id}
                      onClick={() => setActiveTab(tab.id)}
                      className={`py-4 px-6 text-sm font-medium whitespace-nowrap flex-shrink-0 ${
                        activeTab === tab.id
                          ? 'text-blue-600 bg-blue-50 rounded-t-md'
                          : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50 rounded-t-md'
                      }`}
                    >
                      {tab.label}
                      <span className={`ml-2 py-0.5 px-2 rounded-full text-xs ${
                        activeTab === tab.id
                          ? 'bg-blue-100 text-blue-600'
                          : 'bg-gray-100 text-gray-500'
                      }`}>
                        {formatTabCount(tab.id)}
                      </span>
                    </button>
                  ))}
                </nav>

                {/* Filter Input and Export - fixed on the right */}
                <div className="py-3 flex items-center gap-3 flex-shrink-0 ml-4">
                  <div className="relative">
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                      <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                      </svg>
                    </div>
                    <input
                      type="text"
                      placeholder="Filter..."
                      value={filter}
                      onChange={(e) => setFilter(e.target.value)}
                      className="block w-64 pl-9 pr-3 py-2 border border-gray-300 rounded-md text-sm placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                    />
                    {filter && (
                      <button
                        onClick={() => setFilter('')}
                        className="absolute inset-y-0 right-0 pr-3 flex items-center"
                      >
                        <svg className="h-4 w-4 text-gray-400 hover:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    )}
                  </div>
                  <button
                    onClick={exportToCSV}
                    className="px-3 py-2 text-sm bg-green-600 text-white rounded-md hover:bg-green-700 flex items-center gap-1"
                  >
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                    Export CSV
                  </button>
                </div>
              </div>
            </div>

            {/* Tab Content */}
            <div>
              {activeTab === 'accounts' && (
                <CostTable accounts={filteredData?.accounts} />
              )}
              {activeTab === 'regions' && (
                <CostTable regions={filteredData?.regions} />
              )}
              {activeTab === 'ec2' && (
                <CostTable ec2={filteredData?.ec2} />
              )}
              {activeTab === 'ebs' && (
                <CostTable ebs={filteredData?.ebs} />
              )}
              {activeTab === 'ecs' && (
                <CostTable ecs={filteredData?.ecs} />
              )}
              {activeTab === 'rds' && (
                <CostTable rds={filteredData?.rds} />
              )}
              {activeTab === 'eks' && (
                <CostTable eks={filteredData?.eks} />
              )}
              {activeTab === 'elb' && (
                <CostTable elb={filteredData?.elb} />
              )}
              {activeTab === 'nat' && (
                <CostTable nat={filteredData?.nat} />
              )}
              {activeTab === 'eip' && (
                <CostTable eip={filteredData?.eip} />
              )}
              {activeTab === 'secrets' && (
                <CostTable secrets={filteredData?.secrets} />
              )}
              {activeTab === 'publicipv4' && (
                <CostTable publicipv4={filteredData?.publicipv4} />
              )}
            </div>
          </div>
        </>
      )}

    </div>
  );
};

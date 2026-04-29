import React, { useEffect, useRef, useState } from 'react';
import { useAppSelector, useAppDispatch } from '../../hooks/useAppDispatch';
import {
  setSelectedAccounts,
  setSelectedRegions,
  setSelectedResources,
  fetchCosts,
  cancelCostLoad,
  clearCache,
} from '../../store/costSlice';
import { MultiSelectDropdown } from '../common/MultiSelectDropdown';
import { RESOURCE_TYPES } from '../../types/cost';
import { loadAppConfig, shouldExcludeAccount, shouldExcludeRegion } from '../../services/configService';

type LoadPromise = Promise<unknown> & { abort: () => void };

export const ResourceSelector: React.FC = () => {
  const dispatch = useAppDispatch();
  const {
    config,
    configLoading,
    configError,
    selectedAccounts,
    selectedRegions,
    selectedResources,
    loading,
    clearingCache,
    hasLoadedData,
  } = useAppSelector((state) => state.costs);
  const [appConfig, setAppConfig] = useState<Awaited<ReturnType<typeof loadAppConfig>>>({});
  const [loadMenuOpen, setLoadMenuOpen] = useState(false);
  const activeLoadRef = useRef<LoadPromise | null>(null);
  const loadMenuRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    loadAppConfig().then(setAppConfig);
  }, []);

  useEffect(() => {
    if (!loadMenuOpen) return;

    const handleDocumentClick = (event: MouseEvent) => {
      if (loadMenuRef.current?.contains(event.target as Node)) {
        return;
      }
      setLoadMenuOpen(false);
    };
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setLoadMenuOpen(false);
      }
    };

    document.addEventListener('click', handleDocumentClick);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('click', handleDocumentClick);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [loadMenuOpen]);

  const handleAccountChange = (selected: string[]) => {
    dispatch(setSelectedAccounts(selected));
  };

  const handleRegionChange = (selected: string[]) => {
    dispatch(setSelectedRegions(selected));
  };

  const handleResourceChange = (selected: string[]) => {
    dispatch(setSelectedResources(selected));
  };

  const startCostLoad = () => {
    const promise = dispatch(fetchCosts()) as LoadPromise;
    activeLoadRef.current = promise;
    promise.finally(() => {
      if (activeLoadRef.current === promise) {
        activeLoadRef.current = null;
      }
    });
  };

  const handleLoadData = () => {
    setLoadMenuOpen(false);
    startCostLoad();
  };

  const handleClearCacheAndReload = async () => {
    if (!canClearCacheAndReload) {
      return;
    }
    setLoadMenuOpen(false);
    try {
      await dispatch(clearCache()).unwrap();
    } catch {
      return;
    }
    startCostLoad();
  };

  const handleCancelLoad = () => {
    activeLoadRef.current?.abort();
    activeLoadRef.current = null;
    dispatch(cancelCostLoad());
  };

  const canLoad = selectedAccounts.length > 0 && selectedRegions.length > 0 && selectedResources.length > 0;
  const canClearCacheAndReload = hasLoadedData && canLoad && !clearingCache && !loading;

  if (configLoading) {
    return (
      <div className="bg-white shadow rounded-lg p-6 mb-6">
        <div className="flex items-center justify-center h-24">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <span className="ml-3 text-gray-600">Loading available accounts and regions...</span>
        </div>
      </div>
    );
  }

  if (configError) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-6">
        <div className="flex">
          <div className="flex-shrink-0">
            <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                clipRule="evenodd"
              />
            </svg>
          </div>
          <div className="ml-3">
            <h3 className="text-sm font-medium text-red-800">Error loading configuration</h3>
            <p className="mt-1 text-sm text-red-700">{configError}</p>
          </div>
        </div>
      </div>
    );
  }

  if (!config) {
    return null;
  }

  const accountOptions = [...config.accounts]
    .filter((account) => !shouldExcludeAccount(account, appConfig))
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((account) => ({
      value: account.name,
      label: account.id && account.name !== account.id ? ` ${account.name} (${account.id})` : account.name,
    }));

  const regionOptions = [...config.regions]
    .filter((region) => !shouldExcludeRegion(region, appConfig))
    .sort((a, b) => a.localeCompare(b))
    .map((region) => ({
      value: region,
      label: region,
    }));

  const resourceLabels: Record<string, string> = {
    ec2: 'EC2 Instances',
    ebs: 'EBS Volumes',
    ecs: 'ECS Clusters',
    rds: 'RDS Instances',
    eks: 'EKS Clusters',
    elb: 'Load Balancers',
    nat: 'NAT Gateways',
    eip: 'Elastic IPs',
    secrets: 'Secrets',
    publicipv4: 'Public IPv4 Addrs',
    lambda: 'Lambda Functions',
  };

  const resourceOptions = RESOURCE_TYPES.map((resource) => ({
    value: resource,
    label: resourceLabels[resource] || resource.toUpperCase(),
  })).sort((a, b) => a.label.localeCompare(b.label));

  return (
    <div className="bg-white shadow rounded-lg p-6 mb-6">
      <h2 className="text-lg font-medium text-gray-900 mb-4">Select Resources to Query</h2>
      {/* fr vals below act as a percentage of total width. Like 40%/20%/20%/20% */}
      <div className="grid grid-cols-1 md:grid-cols-[4fr_2fr_2fr_2fr] gap-4">
        {/* Account Selector */}
        <MultiSelectDropdown
          id="accounts"
          label="Accounts"
          options={accountOptions}
          selected={selectedAccounts}
          onChange={handleAccountChange}
          placeholder="Select accounts..."
        />

        {/* Region Selector */}
        <MultiSelectDropdown
          id="regions"
          label="Regions"
          options={regionOptions}
          selected={selectedRegions}
          onChange={handleRegionChange}
          placeholder="Select regions..."
        />

        {/* Resource Type Selector */}
        <MultiSelectDropdown
          id="resources"
          label="Resources"
          options={resourceOptions}
          selected={selectedResources}
          onChange={handleResourceChange}
          placeholder="Select resources..."
        />

        {/* Actions */}
        <div className="flex items-end">
          {loading ? (
            <div className="relative w-full">
              <button
                onClick={handleCancelLoad}
                className="w-full px-3 py-2.5 text-sm font-medium bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 flex items-center justify-center gap-2"
              >
                <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M6 6h8v8H6z" />
                </svg>
                Stop
              </button>
            </div>
          ) : (
            <div className="relative w-full">
              <button
                onClick={handleLoadData}
                onContextMenu={(event) => {
                  event.preventDefault();
                  if (canLoad && !clearingCache) {
                    setLoadMenuOpen(true);
                  }
                }}
                disabled={!canLoad || clearingCache}
                className="w-full px-3 py-2.5 text-sm font-medium bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
              >
                {clearingCache ? (
                  <>
                    <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path
                        className="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      />
                    </svg>
                    Clearing Cache...
                  </>
                ) : (
                  <>
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                      />
                    </svg>
                    Load Data
                  </>
                )}
              </button>
              {loadMenuOpen && (
                <div
                  ref={loadMenuRef}
                  className="absolute left-1/2 top-full z-20 -mt-1 w-52 -translate-x-1/2 rounded-md border border-gray-200 bg-white p-1 shadow-lg"
                >
                  <button
                    type="button"
                    onClick={handleClearCacheAndReload}
                    disabled={!canClearCacheAndReload}
                    className="flex w-full items-center justify-center rounded px-3 py-2 text-center text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:text-gray-400 disabled:hover:bg-white"
                  >
                    Clear Cache and Reload
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {!canLoad && (
        <p className="mt-4 text-sm text-amber-600">
          Please select at least one account, region, and resource type to load cost data.
        </p>
      )}
    </div>
  );
};

import type { CostResponse, CostFilters, ConfigResponse } from '../types/cost';

async function fetchApi<T>(url: string): Promise<T> {
  const response = await fetch(`/api/v1${url}`, {
    signal: AbortSignal.timeout(5 * 60 * 1000), // 5 minutes for large queries
  });
  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }
  return response.json();
}

export const costApi = {
  async getCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    if (filters.resources?.length) {
      params.set('resource', filters.resources.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs?${params.toString()}`);
    return response;
  },

  async getAccountCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/accounts?${params.toString()}`);
    return response;
  },

  async getRegionCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/regions?${params.toString()}`);
    return response;
  },

  async getEC2Costs(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/ec2?${params.toString()}`);
    return response;
  },

  async getEBSCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/ebs?${params.toString()}`);
    return response;
  },

  async getRDSCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/rds?${params.toString()}`);
    return response;
  },

  async getEKSCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/eks?${params.toString()}`);
    return response;
  },

  async getELBCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/elb?${params.toString()}`);
    return response;
  },

  async getNATGatewayCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/nat?${params.toString()}`);
    return response;
  },

  async getElasticIPCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/eip?${params.toString()}`);
    return response;
  },

  async getSecretsCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/secrets?${params.toString()}`);
    return response;
  },

  async getPublicIPv4Costs(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await fetchApi<CostResponse>(`/costs/publicipv4?${params.toString()}`);
    return response;
  },
};

export const configApi = {
  async getConfig(): Promise<ConfigResponse> {
    return fetchApi<ConfigResponse>('/config');
  },
};

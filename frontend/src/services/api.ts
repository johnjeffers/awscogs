import axios from 'axios';
import type { CostResponse, CostFilters, ConfigResponse } from '../types/cost';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 5 * 60 * 1000, // 5 minutes for large queries
});

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
    const response = await api.get<CostResponse>(`/costs?${params.toString()}`);
    return response.data;
  },

  async getAccountCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/accounts?${params.toString()}`);
    return response.data;
  },

  async getRegionCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/regions?${params.toString()}`);
    return response.data;
  },

  async getEC2Costs(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/ec2?${params.toString()}`);
    return response.data;
  },

  async getEBSCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/ebs?${params.toString()}`);
    return response.data;
  },

  async getRDSCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/rds?${params.toString()}`);
    return response.data;
  },

  async getEKSCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/eks?${params.toString()}`);
    return response.data;
  },

  async getELBCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/elb?${params.toString()}`);
    return response.data;
  },

  async getNATGatewayCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/nat?${params.toString()}`);
    return response.data;
  },

  async getElasticIPCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/eip?${params.toString()}`);
    return response.data;
  },

  async getSecretsCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.accounts?.length) {
      params.set('account', filters.accounts.join(','));
    }
    if (filters.regions?.length) {
      params.set('region', filters.regions.join(','));
    }
    const response = await api.get<CostResponse>(`/costs/secrets?${params.toString()}`);
    return response.data;
  },
};

export const configApi = {
  async getConfig(): Promise<ConfigResponse> {
    const response = await api.get<ConfigResponse>('/config');
    return response.data;
  },
};

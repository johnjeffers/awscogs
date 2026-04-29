import type { CostResponse, CostFilters, ConfigResponse } from '../types/cost';

function createTimeoutSignal(timeoutMs: number, signal?: AbortSignal): AbortSignal {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), timeoutMs);

  const abort = () => controller.abort();
  if (signal?.aborted) {
    controller.abort();
  } else {
    signal?.addEventListener('abort', abort, { once: true });
  }
  controller.signal.addEventListener(
    'abort',
    () => {
      window.clearTimeout(timeout);
      signal?.removeEventListener('abort', abort);
    },
    { once: true },
  );

  return controller.signal;
}

function buildCostParams(filters: CostFilters = {}): URLSearchParams {
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
  return params;
}

function appendRequestId(params: URLSearchParams, requestId?: string) {
  if (requestId) {
    params.set('_rid', requestId);
  }
}

async function fetchApi<T>(url: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`/api/v1${url}`, {
    signal: createTimeoutSignal(5 * 60 * 1000, signal), // 5 minutes for large queries
  });
  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }
  return response.json();
}

async function postApi<T>(url: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`/api/v1${url}`, {
    method: 'POST',
    signal: createTimeoutSignal(5 * 60 * 1000, signal),
  });
  if (response.status === 404) {
    return await fetchApi<T>(url, signal);
  }
  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }
  return response.json();
}

export const costApi = {
  async getCosts(filters: CostFilters = {}, signal?: AbortSignal, requestId?: string): Promise<CostResponse> {
    const params = buildCostParams(filters);
    appendRequestId(params, requestId);
    return await fetchApi<CostResponse>(`/costs?${params.toString()}`, signal);
  },

  async getAccountCosts(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/accounts?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getRegionCosts(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/regions?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getEC2Costs(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/ec2?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getEBSCosts(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/ebs?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getRDSCosts(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/rds?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getEKSCosts(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/eks?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getELBCosts(
    filters: CostFilters = {},
    options?: { includeUsage?: boolean; usageWindow?: string },
    signal?: AbortSignal,
  ): Promise<CostResponse> {
    const params = buildCostParams(filters);
    if (options?.includeUsage) {
      params.set('includeUsage', 'true');
      if (options.usageWindow) {
        params.set('usageWindow', options.usageWindow);
      }
    }
    const response = await fetchApi<CostResponse>(`/costs/elb?${params.toString()}`, signal);
    return response;
  },

  async getNATGatewayCosts(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/nat?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getElasticIPCosts(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/eip?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getSecretsCosts(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/secrets?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getPublicIPv4Costs(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/publicipv4?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async getLambdaCosts(filters: CostFilters = {}, signal?: AbortSignal): Promise<CostResponse> {
    const response = await fetchApi<CostResponse>(`/costs/lambda?${buildCostParams(filters).toString()}`, signal);
    return response;
  },

  async clearCache(): Promise<{ status: string }> {
    return await postApi<{ status: string }>('/cache/clear');
  },
};

export const configApi = {
  async getConfig(): Promise<ConfigResponse> {
    return fetchApi<ConfigResponse>('/config');
  },
};

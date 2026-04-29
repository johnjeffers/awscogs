import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';
import type { CostResponse, CostFilters, ConfigResponse } from '../types/cost';
import { costApi, configApi } from '../services/api';

interface CostState {
  data: CostResponse | null;
  config: ConfigResponse | null;
  configLoading: boolean;
  configError: string | null;
  loading: boolean;
  clearingCache: boolean;
  error: string | null;
  filters: CostFilters;
  selectedAccounts: string[];
  selectedRegions: string[];
  selectedResources: string[];
  hasLoadedData: boolean;
  dataVersion: number;
  currentRequestId: string | null;
}

const initialState: CostState = {
  data: null,
  config: null,
  configLoading: false,
  configError: null,
  loading: false,
  clearingCache: false,
  error: null,
  filters: {},
  selectedAccounts: [],
  selectedRegions: [],
  selectedResources: [],
  hasLoadedData: false,
  dataVersion: 0,
  currentRequestId: null,
};

export const fetchCosts = createAsyncThunk('costs/fetch', async (_, { getState, requestId, signal }) => {
  const state = getState() as { costs: CostState };
  const filters: CostFilters = {
    accounts: state.costs.selectedAccounts.length > 0 ? state.costs.selectedAccounts : undefined,
    regions: state.costs.selectedRegions.length > 0 ? state.costs.selectedRegions : undefined,
    resources: state.costs.selectedResources.length > 0 ? state.costs.selectedResources : undefined,
  };
  return await costApi.getCosts(filters, signal, requestId);
});

export const fetchELBUsage = createAsyncThunk('costs/fetchELBUsage', async (usageWindow: string, { getState }) => {
  const state = getState() as { costs: CostState };
  const filters: CostFilters = {
    accounts: state.costs.selectedAccounts.length > 0 ? state.costs.selectedAccounts : undefined,
    regions: state.costs.selectedRegions.length > 0 ? state.costs.selectedRegions : undefined,
  };
  return await costApi.getELBCosts(filters, { includeUsage: true, usageWindow });
});

export const fetchConfig = createAsyncThunk('costs/fetchConfig', async () => {
  return await configApi.getConfig();
});

export const clearCache = createAsyncThunk('costs/clearCache', async () => {
  return await costApi.clearCache();
});

const costSlice = createSlice({
  name: 'costs',
  initialState,
  reducers: {
    setSelectedAccounts: (state, action: PayloadAction<string[]>) => {
      state.selectedAccounts = action.payload;
    },
    setSelectedRegions: (state, action: PayloadAction<string[]>) => {
      state.selectedRegions = action.payload;
    },
    setSelectedResources: (state, action: PayloadAction<string[]>) => {
      state.selectedResources = action.payload;
    },
    clearELBUsage: (state) => {
      if (state.data?.loadBalancers) {
        for (const lb of state.data.loadBalancers) {
          lb.usageStatus = undefined;
          lb.usageError = undefined;
          lb.requestVolume = undefined;
          lb.bandwidthBytes = undefined;
          lb.requestMetricName = undefined;
          lb.bandwidthMetricName = undefined;
          lb.usageWindow = undefined;
          lb.usageStart = undefined;
          lb.usageEnd = undefined;
        }
      }
    },
    cancelCostLoad: (state) => {
      state.loading = false;
      state.currentRequestId = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchCosts.pending, (state, action) => {
        state.loading = true;
        state.error = null;
        state.currentRequestId = action.meta.requestId;
      })
      .addCase(fetchCosts.fulfilled, (state, action) => {
        if (state.currentRequestId !== action.meta.requestId) {
          return;
        }
        state.loading = false;
        state.currentRequestId = null;
        state.data = action.payload;
        state.hasLoadedData = true;
        state.dataVersion += 1;
      })
      .addCase(fetchCosts.rejected, (state, action) => {
        if (state.currentRequestId !== action.meta.requestId) {
          return;
        }
        state.loading = false;
        state.currentRequestId = null;
        if (!action.meta.aborted) {
          state.error = action.error.message || 'Failed to fetch costs';
        }
      })
      .addCase(fetchELBUsage.fulfilled, (state, action) => {
        if (state.data) {
          state.data.loadBalancers = action.payload.loadBalancers;
        }
      })
      .addCase(clearCache.pending, (state) => {
        state.clearingCache = true;
        state.error = null;
      })
      .addCase(clearCache.fulfilled, (state) => {
        state.clearingCache = false;
      })
      .addCase(clearCache.rejected, (state, action) => {
        state.clearingCache = false;
        state.error = action.error.message || 'Failed to clear cache';
      })
      .addCase(fetchConfig.pending, (state) => {
        state.configLoading = true;
        state.configError = null;
      })
      .addCase(fetchConfig.fulfilled, (state, action) => {
        state.configLoading = false;
        state.config = action.payload;
      })
      .addCase(fetchConfig.rejected, (state, action) => {
        state.configLoading = false;
        state.configError = action.error.message || 'Failed to fetch configuration';
      });
  },
});

export const { setSelectedAccounts, setSelectedRegions, setSelectedResources, clearELBUsage, cancelCostLoad } =
  costSlice.actions;
export default costSlice.reducer;

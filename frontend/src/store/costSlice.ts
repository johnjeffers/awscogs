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
  error: string | null;
  filters: CostFilters;
  selectedAccounts: string[];
  selectedRegions: string[];
  selectedResources: string[];
  hasLoadedData: boolean;
}

const initialState: CostState = {
  data: null,
  config: null,
  configLoading: false,
  configError: null,
  loading: false,
  error: null,
  filters: {},
  selectedAccounts: [],
  selectedRegions: [],
  selectedResources: [],
  hasLoadedData: false,
};

export const fetchCosts = createAsyncThunk(
  'costs/fetch',
  async (_, { getState }) => {
    const state = getState() as { costs: CostState };
    const filters: CostFilters = {
      accounts: state.costs.selectedAccounts.length > 0 ? state.costs.selectedAccounts : undefined,
      regions: state.costs.selectedRegions.length > 0 ? state.costs.selectedRegions : undefined,
      resources: state.costs.selectedResources.length > 0 ? state.costs.selectedResources : undefined,
    };
    return await costApi.getCosts(filters);
  }
);

export const fetchConfig = createAsyncThunk(
  'costs/fetchConfig',
  async () => {
    return await configApi.getConfig();
  }
);

const costSlice = createSlice({
  name: 'costs',
  initialState,
  reducers: {
    setFilters: (state, action: PayloadAction<CostFilters>) => {
      state.filters = action.payload;
    },
    setSelectedAccounts: (state, action: PayloadAction<string[]>) => {
      state.selectedAccounts = action.payload;
    },
    setSelectedRegions: (state, action: PayloadAction<string[]>) => {
      state.selectedRegions = action.payload;
    },
    setSelectedResources: (state, action: PayloadAction<string[]>) => {
      state.selectedResources = action.payload;
    },
    clearError: (state) => {
      state.error = null;
    },
    clearData: (state) => {
      state.data = null;
      state.hasLoadedData = false;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchCosts.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchCosts.fulfilled, (state, action) => {
        state.loading = false;
        state.data = action.payload;
        state.hasLoadedData = true;
      })
      .addCase(fetchCosts.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch costs';
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

export const { setFilters, setSelectedAccounts, setSelectedRegions, setSelectedResources, clearError, clearData } = costSlice.actions;
export default costSlice.reducer;

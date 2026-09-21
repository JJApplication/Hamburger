"use client";

import { create } from "zustand";

import { fetchGatewayData, fetchManagementDomains } from "@/lib/api/gateway";
import type { DomainConnection, ExperimentFeature, GatewayData } from "@/types/gateway";

interface DomainFilter {
  keyword: string;
  status: "all" | "online" | "offline" | "warning" | "unknown";
  port?: number;
}

interface GatewayStoreState {
  data: GatewayData | null;
  isLoading: boolean;
  error: string | null;
  domainFilter: DomainFilter;
  initialize: (token: string) => Promise<void>;
  refresh: (token?: string) => Promise<void>;
  refreshDomains: (token: string) => Promise<void>;
  setDomainFilter: (patch: Partial<DomainFilter>) => void;
  toggleExperiment: (key: ExperimentFeature["key"]) => void;
}

const initialFilter: DomainFilter = {
  keyword: "",
  status: "all",
};

const EMPTY_DOMAINS: DomainConnection[] = [];
const EMPTY_EXPERIMENTS: ExperimentFeature[] = [];

export const useGatewayStore = create<GatewayStoreState>((set, get) => ({
  data: null,
  isLoading: false,
  error: null,
  domainFilter: initialFilter,
  initialize: async (token) => {
    if (get().data) {
      return;
    }
    await get().refresh(token);
  },
  refresh: async (token) => {
    if (!token) {
      set({ data: null, error: null, isLoading: false });
      return;
    }
    set({ isLoading: true, error: null });
    try {
      const data = await fetchGatewayData(token);
      set({ data, isLoading: false, error: null });
    } catch (reason) {
      const status = reason instanceof Error ? (reason as Error & { status?: number }).status : undefined;
      set({
        data: status === 401 ? null : get().data,
        isLoading: false,
        error: "数据加载失败，请稍后重试。",
      });
    }
  },
  refreshDomains: async (token) => {
    if (!token) return;
    if (!get().data) {
      await get().refresh(token);
      return;
    }
    try {
      const result = await fetchManagementDomains(token);
      set((state) => state.data ? { data: { ...state.data, domains: result.domains ?? [] } } : state);
    } catch (reason) {
      const status = reason instanceof Error ? (reason as Error & { status?: number }).status : undefined;
      if (status === 401) set({ data: null });
    }
  },
  setDomainFilter: (patch) => {
    set((state) => ({
      domainFilter: {
        ...state.domainFilter,
        ...patch,
      },
    }));
  },
  toggleExperiment: (key) => {
    set((state) => {
      if (!state.data) {
        return state;
      }
      return {
        data: {
          ...state.data,
          experiments: state.data.experiments.map((feature) =>
            feature.key === key
              ? {
                  ...feature,
                  enabled: !feature.enabled,
                }
              : feature,
          ),
        },
      };
    });
  },
}));

const matchFilter = (domain: DomainConnection, filter: DomainFilter): boolean => {
  const keyword = filter.keyword.trim().toLowerCase();
  const byKeyword =
    keyword.length === 0 ||
    domain.domain.toLowerCase().includes(keyword) ||
    domain.mappings.some((mapping) => mapping.backendTarget.toLowerCase().includes(keyword));

  const byStatus = filter.status === "all" || domain.status === filter.status;
  const byPort = filter.port == null || domain.mappings.some((mapping) => mapping.frontendPort === filter.port);

  return byKeyword && byStatus && byPort;
};

const createFilteredDomainsSelector = () => {
  let prevDomains: DomainConnection[] | null = null;
  let prevFilter: DomainFilter | null = null;
  let prevResult: DomainConnection[] = EMPTY_DOMAINS;

  return (state: GatewayStoreState) => {
    const domains = state.data?.domains;
    if (!domains) {
      prevDomains = null;
      prevFilter = state.domainFilter;
      prevResult = EMPTY_DOMAINS;
      return EMPTY_DOMAINS;
    }

    if (domains === prevDomains && state.domainFilter === prevFilter) {
      return prevResult;
    }

    prevDomains = domains;
    prevFilter = state.domainFilter;
    prevResult = domains.filter((domain) => matchFilter(domain, state.domainFilter));
    return prevResult;
  };
};

const selectFilteredDomains = createFilteredDomainsSelector();

export const useGatewaySelectors = {
  overviewSummary: (state: GatewayStoreState) => state.data?.overview,
  configs: (state: GatewayStoreState) => state.data?.configs,
  experiments: (state: GatewayStoreState) => state.data?.experiments ?? EMPTY_EXPERIMENTS,
  filteredDomains: selectFilteredDomains,
};

"use client";

import { useEffect, useMemo, useState } from "react";

import { DataState } from "@/components/common/data-state";
import { Panel } from "@/components/common/panel";
import { DomainCard } from "@/components/domains/domain-card";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { useGatewayBootstrap } from "@/lib/hooks/use-gateway-bootstrap";
import { usePreferences } from "@/lib/preferences/preferences-context";
import { useGatewaySelectors, useGatewayStore } from "@/store/gateway-store";

const PAGE_SIZE = 5;

export default function DomainsPage() {
  useGatewayBootstrap();
  const { t } = usePreferences();

  const domains = useGatewayStore(useGatewaySelectors.filteredDomains);
  const filter = useGatewayStore((state) => state.domainFilter);
  const isLoading = useGatewayStore((state) => state.isLoading);
  const error = useGatewayStore((state) => state.error);
  const refresh = useGatewayStore((state) => state.refresh);
  const setDomainFilter = useGatewayStore((state) => state.setDomainFilter);
  const [currentPage, setCurrentPage] = useState(1);

  const portFilter = useMemo(() => (filter.port == null ? "" : String(filter.port)), [filter.port]);
  const totalPages = Math.max(1, Math.ceil(domains.length / PAGE_SIZE));
  const pagedDomains = useMemo(
    () => domains.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE),
    [domains, currentPage],
  );
  const pageNumbers = useMemo(() => {
    const start = Math.max(1, currentPage - 2);
    const end = Math.min(totalPages, start + 4);
    const normalizedStart = Math.max(1, end - 4);
    return Array.from({ length: end - normalizedStart + 1 }, (_, index) => normalizedStart + index);
  }, [currentPage, totalPages]);

  useEffect(() => {
    setCurrentPage(1);
  }, [filter.keyword, filter.status, filter.port]);

  useEffect(() => {
    if (currentPage > totalPages) {
      setCurrentPage(totalPages);
    }
  }, [currentPage, totalPages]);

  return (
    <DashboardShell title={t("domains.title")} subtitle={t("domains.subtitle")}>
      <Panel className="grid gap-3 md:grid-cols-3">
        <input
          value={filter.keyword}
          onChange={(event) => setDomainFilter({ keyword: event.target.value })}
          placeholder={t("domains.search")}
          className="panel-strong text-primary placeholder:text-tertiary rounded-xl border border-[var(--border)] px-3 py-2 text-sm outline-none ring-0 focus:border-cyan-400/60"
        />
        <select
          value={filter.status}
          onChange={(event) => setDomainFilter({ status: event.target.value as typeof filter.status })}
          className="panel-strong text-primary rounded-xl border border-[var(--border)] px-3 py-2 text-sm outline-none focus:border-cyan-400/60"
        >
          <option value="all">{t("domains.filterAll")}</option>
          <option value="online">{t("domains.filterOnline")}</option>
          <option value="warning">{t("domains.filterWarning")}</option>
          <option value="offline">{t("domains.filterOffline")}</option>
        </select>
        <input
          value={portFilter}
          onChange={(event) => {
            const value = event.target.value.trim();
            setDomainFilter({ port: value ? Number(value) : undefined });
          }}
          placeholder={t("domains.filterPort")}
          className="panel-strong text-primary placeholder:text-tertiary rounded-xl border border-[var(--border)] px-3 py-2 text-sm outline-none ring-0 focus:border-cyan-400/60"
        />
      </Panel>

      <DataState isLoading={isLoading && domains.length === 0} error={error} onRetry={() => void refresh()} />
      {domains.length === 0 && !isLoading && !error ? (
        <Panel className="text-secondary py-12 text-center text-sm">{t("domains.empty")}</Panel>
      ) : null}

      <section className="grid gap-4 2xl:grid-cols-2">
        {pagedDomains.map((domain) => (
          <DomainCard key={domain.domain} domain={domain} />
        ))}
      </section>

      {domains.length > 0 ? (
        <Panel className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-secondary text-sm">
            {t("domains.paginationSummary", { page: currentPage, total: totalPages, count: domains.length })}
          </p>
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={currentPage <= 1}
              onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
              className="text-primary rounded-lg border border-[var(--border)] px-3 py-1.5 text-sm transition disabled:opacity-40"
            >
              {t("domains.prev")}
            </button>
            {pageNumbers.map((page) => {
              return (
                <button
                  key={page}
                  type="button"
                  onClick={() => setCurrentPage(page)}
                  className={[
                    "rounded-lg border px-3 py-1.5 text-sm transition",
                    page === currentPage
                      ? "border-emerald-400/60 bg-emerald-500/15 text-emerald-300"
                      : "text-primary border-[var(--border)] hover:opacity-85",
                  ].join(" ")}
                >
                  {page}
                </button>
              );
            })}
            <button
              type="button"
              disabled={currentPage >= totalPages}
              onClick={() => setCurrentPage((page) => Math.min(totalPages, page + 1))}
              className="text-primary rounded-lg border border-[var(--border)] px-3 py-1.5 text-sm transition disabled:opacity-40"
            >
              {t("domains.next")}
            </button>
          </div>
        </Panel>
      ) : null}
    </DashboardShell>
  );
}

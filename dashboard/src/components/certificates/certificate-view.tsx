"use client";

import { useMemo, useState } from "react";

import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import type { CertificateCheck, CertificateModel, CertificateStatus, KeyMatch } from "@/lib/certificates/load";
import { usePreferences, type MessageKey } from "@/lib/preferences/preferences-context";

const statusLabel: Record<CertificateStatus, MessageKey> = {
  valid: "certificates.ok",
  expiring: "certificates.expiringSoon",
  expired: "certificates.expired",
  missing: "certificates.fileMissing",
  invalid: "certificates.invalid",
  key: "certificates.keyProblem",
};

const keyLabel: Record<Exclude<KeyMatch, "unchecked">, MessageKey> = {
  matched: "certificates.keyMatched",
  mismatch: "certificates.keyMismatch",
  missing: "certificates.keyMissing",
  unread: "certificates.keyUnread",
};

function statusClass(status: CertificateStatus): string {
  if (status === "valid") return "border-emerald-400/40 bg-emerald-500/15 text-emerald-300 [html.theme-light_&]:text-emerald-700";
  if (status === "expiring") return "border-amber-400/40 bg-amber-500/15 text-amber-300 [html.theme-light_&]:text-amber-700";
  return "border-rose-400/40 bg-rose-500/15 text-rose-300 [html.theme-light_&]:text-rose-700";
}

function searchText(certificate: CertificateCheck): string {
  const names = certificate.chain.flatMap((item) => [item.commonName, item.subject, item.issuer, ...item.subjectAltNames]);
  return [certificate.name, certificate.server, certificate.certFile, certificate.keyFile, ...certificate.domains.map((domain) => domain.name), ...names].join(" ");
}

export function CertificateView({ model }: { model: CertificateModel }) {
  const { t } = usePreferences();
  const [keyword, setKeyword] = useState("");
  const query = keyword.trim().toLowerCase();
  const visible = useMemo(
    () => model.certificates.filter((certificate) => !query || searchText(certificate).toLowerCase().includes(query)),
    [model.certificates, query],
  );
  const counts = {
    valid: model.certificates.filter((certificate) => certificate.status === "valid").length,
    expiring: model.certificates.filter((certificate) => certificate.status === "expiring").length,
    problem: model.certificates.filter((certificate) => certificate.status !== "valid" && certificate.status !== "expiring").length,
  };

  return (
    <DashboardShell title={t("certificates.title")} subtitle={t("certificates.subtitle")}>
      <Panel className="space-y-4">
        <div>
          <p className="text-secondary text-xs">{t("domainConfig.source")} {model.source}</p>
          <p className="text-tertiary mt-1 text-xs">{t("certificates.checkedAt", { time: model.checkedAt })}</p>
        </div>
        {model.error ? <p className="rounded-xl border border-rose-400/40 bg-rose-500/10 px-3 py-2 text-sm text-rose-300 [html.theme-light_&]:text-rose-700">{model.error}</p> : null}
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Stat label={t("certificates.total")} value={model.certificates.length} />
          <Stat label={t("certificates.valid")} value={counts.valid} />
          <Stat label={t("certificates.expiring")} value={counts.expiring} />
          <Stat label={t("certificates.problem")} value={counts.problem} />
        </div>
      </Panel>

      <Panel>
        <input
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          placeholder={t("certificates.search")}
          className="panel-strong text-primary placeholder:text-tertiary w-full rounded-xl border border-[var(--border)] px-3 py-2 text-sm outline-none focus:border-emerald-400/70 lg:max-w-sm"
        />
      </Panel>

      {visible.length === 0 ? <Panel><p className="text-secondary text-sm">{model.certificates.length === 0 ? t("certificates.empty") : t("certificates.noMatch")}</p></Panel> : (
        <div className="space-y-4">
          {visible.map((certificate) => {
            const leaf = certificate.chain[0];
            return (
              <article key={certificate.id} className="panel space-y-4 rounded-2xl border p-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h2 className="break-all text-lg font-semibold">{certificate.name}</h2>
                    <p className="text-tertiary mt-1 text-xs">
                      {[certificate.server, certificate.minVersion, certificate.autoTls === null ? "" : t(certificate.autoTls ? "certificates.autoTlsOn" : "certificates.autoTlsOff")].filter(Boolean).join(" · ")}
                    </p>
                  </div>
                  <span className={`inline-flex rounded-full border px-2 py-0.5 text-xs font-medium ${statusClass(certificate.status)}`}>{t(statusLabel[certificate.status])}</span>
                </div>

                <div className="grid gap-2 md:grid-cols-2">
                  <p className="text-secondary break-all font-mono text-xs"><span className="text-tertiary">{t("certificates.certFile")} </span>{certificate.certFile}</p>
                  <p className="text-secondary break-all font-mono text-xs">
                    <span className="text-tertiary">{t("certificates.keyFile")} </span>
                    {certificate.keyFile || "—"}
                    {certificate.keyMatch !== "unchecked" ? ` · ${t(keyLabel[certificate.keyMatch])}` : ""}
                  </p>
                </div>
                {certificate.error ? <p className="text-sm text-rose-300 [html.theme-light_&]:text-rose-700">{certificate.error}</p> : null}

                {certificate.domains.length > 0 ? (
                  <div>
                    <p className="text-tertiary text-xs">{t("certificates.domains")}</p>
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {certificate.domains.map((domain) => (
                        <span
                          key={domain.name}
                          className={[
                            "rounded-full px-2 py-0.5 text-xs",
                            domain.covered === false
                              ? "bg-amber-500/15 text-amber-300 [html.theme-light_&]:text-amber-700"
                              : domain.covered
                                ? "bg-emerald-500/15 text-emerald-300 [html.theme-light_&]:text-emerald-700"
                                : "panel-strong text-secondary",
                          ].join(" ")}
                          title={domain.covered === null ? undefined : t(domain.covered ? "certificates.covered" : "certificates.uncovered")}
                        >
                          {domain.name}
                        </span>
                      ))}
                    </div>
                  </div>
                ) : null}

                {leaf ? (
                  <div className="space-y-3">
                    <h3 className="text-sm font-semibold">{t("certificates.chain")}</h3>
                    {certificate.chain.map((item) => (
                      <section key={`${item.serialNumber}-${item.role}`} className="panel-strong rounded-xl px-3 py-3">
                        <p className="text-sm font-medium">{item.commonName}</p>
                        <p className="text-tertiary mt-0.5 text-xs">{t(item.role === "leaf" ? "certificates.leaf" : "certificates.intermediate")}</p>
                        <dl className="mt-3 grid gap-2 md:grid-cols-2">
                          <Field label={t("certificates.subject")} value={item.subject} />
                          <Field label={t("certificates.issuer")} value={item.issuer} />
                          <Field label={t("certificates.sans")} value={item.subjectAltNames.join("、") || t("certificates.noSan")} />
                          <Field label={t("certificates.validFrom")} value={item.validFrom} />
                          <Field label={t("certificates.validTo")} value={`${item.validTo} · ${item.daysRemaining < 0 ? t("certificates.daysAgo", { days: Math.abs(item.daysRemaining) }) : t("certificates.daysLeft", { days: item.daysRemaining })}`} />
                          <Field label={t("certificates.serial")} value={item.serialNumber} />
                          <Field label={t("certificates.fingerprint")} value={item.fingerprint} />
                        </dl>
                      </section>
                    ))}
                  </div>
                ) : null}
              </article>
            );
          })}
        </div>
      )}
    </DashboardShell>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="panel-strong rounded-xl px-3 py-3">
      <p className="text-tertiary text-xs">{label}</p>
      <p className="mt-1 text-2xl font-semibold tabular-nums">{value}</p>
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-tertiary text-xs">{label}</dt>
      <dd className="text-secondary mt-0.5 break-all font-mono text-xs">{value}</dd>
    </div>
  );
}

"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Activity, FlaskConical, Home, Languages, Moon, Network, Settings, Sun, Sandwich } from "lucide-react";

import type { MessageKey } from "@/lib/preferences/preferences-context";
import { usePreferences } from "@/lib/preferences/preferences-context";

interface DashboardShellProps {
  title: string;
  subtitle: string;
  children: React.ReactNode;
}

const navItems: ReadonlyArray<{ href: string; key: MessageKey; icon: React.ComponentType<{ size?: number }> }> = [
  { href: "/", key: "nav.overview", icon: Home },
  { href: "/domains", key: "nav.domains", icon: Network },
  { href: "/configs", key: "nav.configs", icon: Settings },
  { href: "/experiments", key: "nav.experiments", icon: FlaskConical },
];

export function DashboardShell({ title, subtitle, children }: DashboardShellProps) {
  const pathname = usePathname();
  const router = useRouter();
  const { locale, setLocale, theme, setTheme, t } = usePreferences();

  return (
    <div className="app-bg min-h-screen">
      <div className="mx-auto grid max-w-[1600px] grid-cols-1 gap-6 p-4 lg:grid-cols-[248px_1fr] lg:p-6">
        <aside className="panel flex min-h-[calc(100vh-2rem)] flex-col rounded-2xl border p-4 lg:min-h-[calc(100vh-3rem)]">
          <div className="mb-6 flex items-center gap-3 px-2">
            <div className="rounded-xl bg-emerald-500/15 p-2 text-emerald-400">
              <Activity size={20} />
            </div>
            <div>
              <p className="text-secondary text-sm">Hamburger</p>
              <p className="text-lg font-semibold">{t("brand.console")}</p>
            </div>
          </div>
          <nav className="space-y-1">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive = pathname === item.href;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={[
                    "flex items-center gap-3 rounded-xl px-3 py-2 text-sm transition",
                    isActive
                      ? "bg-emerald-500/20 text-emerald-300"
                      : "text-secondary hover:panel-soft hover:text-primary",
                  ].join(" ")}
                >
                  <Icon size={16} />
                  {t(item.key)}
                </Link>
              );
            })}
          </nav>
          <div className="mt-auto flex items-center justify-start gap-2 border-t border-[var(--border)] pt-4">
            <button
              type="button"
              title={t("controls.lang")}
              aria-label={t("controls.lang")}
              onClick={() => setLocale(locale === "zh" ? "en" : "zh")}
              className="panel-soft text-primary flex h-10 w-10 items-center justify-center rounded-full border border-[var(--border)] transition hover:opacity-85"
            >
              <Languages size={16} />
            </button>
            <button
              type="button"
              title={theme === "dark" ? t("controls.themeLight") : t("controls.themeDark")}
              aria-label={theme === "dark" ? t("controls.themeLight") : t("controls.themeDark")}
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
              className="panel-soft text-primary flex h-10 w-10 items-center justify-center rounded-full border border-[var(--border)] transition hover:opacity-85"
            >
              {theme === "dark" ? <Sun size={16} /> : <Moon size={16} />}
            </button>
            <button
              type="button"
              title="Logo"
              aria-label="Logo"
              onClick={() => router.push("/logo")}
              className="panel-soft text-primary flex h-10 w-10 items-center justify-center rounded-full border border-[var(--border)] transition hover:opacity-85"
            >
              <Sandwich size={16} />
            </button>
          </div>
        </aside>

        <main className="space-y-5">
          <header className="panel rounded-2xl border p-5">
            <p className="text-secondary text-sm">{subtitle}</p>
            <h1 className="mt-1 text-2xl font-semibold">{title}</h1>
          </header>
          {children}
        </main>
      </div>
    </div>
  );
}

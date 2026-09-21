"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Activity, FlaskConical, Home, Languages, LogIn, LogOut, Moon, Network, Settings, Sun, Sandwich, UserCircle } from "lucide-react";

import type { MessageKey } from "@/lib/preferences/preferences-context";
import { usePreferences } from "@/lib/preferences/preferences-context";
import { useAuth } from "@/lib/auth/auth-context";
import { LoginDialog } from "@/components/auth/login-dialog";
import { BurgerScene } from "@/components/visuals/burger-scene";

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
  const { user, ready, logout } = useAuth();
  const [loginOpen, setLoginOpen] = useState(false);

  return (
    <div className="app-bg min-h-screen">
      <div className="mx-auto grid max-w-[1600px] grid-cols-1 gap-6 p-4 lg:grid-cols-[248px_1fr] lg:p-6">
        <aside className="panel flex min-h-0 flex-col rounded-2xl border p-4 lg:min-h-[calc(100vh-3rem)]">
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
          <div className="mt-auto border-t border-[var(--border)] pt-4">
            {user ? <div className="mb-3 flex items-center justify-between rounded-xl bg-emerald-500/10 px-3 py-2 text-sm"><span className="flex min-w-0 items-center gap-2"><UserCircle size={17} className="text-emerald-300" /><span className="truncate">{user.nickname || user.username}</span></span><button type="button" onClick={() => void logout()} className="text-secondary rounded-lg p-1.5 hover:text-rose-300" title="退出登录" aria-label="退出登录"><LogOut size={15} /></button></div> : <button type="button" onClick={() => setLoginOpen(true)} className="mb-3 flex w-full items-center justify-center gap-2 rounded-xl bg-emerald-500 px-3 py-2 text-sm font-semibold text-slate-950 transition hover:bg-emerald-400"><LogIn size={16} />登录管理面板</button>}
            <div className="flex items-center justify-start gap-2">
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
          </div>
        </aside>

        <main className="flex min-h-0 flex-col gap-5 lg:min-h-[calc(100vh-3rem)]">
          <header className="panel rounded-2xl border p-5">
            <p className="text-secondary text-sm">{subtitle}</p>
            <h1 className="mt-1 text-2xl font-semibold">{title}</h1>
          </header>
          {!ready ? <section className="panel flex min-h-[560px] items-center justify-center rounded-2xl border lg:min-h-0 lg:flex-1"><p className="text-secondary text-sm">正在恢复登录状态...</p></section> : user ? children : <section className="panel relative flex min-h-[560px] flex-col items-center justify-center overflow-hidden rounded-2xl border bg-[radial-gradient(circle_at_50%_30%,rgba(16,185,129,0.16),transparent_52%)] lg:min-h-0 lg:flex-1"><div className="absolute inset-0 bg-[linear-gradient(135deg,transparent_0%,rgba(255,255,255,0.03)_48%,transparent_100%)]" /><div className="relative z-10 w-full max-w-md"><BurgerScene compact paused={loginOpen} /><div className="mt-8 text-center"><p className="text-secondary text-xs uppercase tracking-[0.3em]">Hamburger Gateway</p><h2 className="mt-2 text-2xl font-semibold">登录后管理你的网关</h2><p className="text-secondary mx-auto mt-2 max-w-md text-sm">统计、域名连接、运行配置和实验能力都会在登录后加载。</p><button type="button" onClick={() => setLoginOpen(true)} className="mt-5 rounded-xl bg-emerald-500 px-5 py-2.5 text-sm font-semibold text-slate-950 hover:bg-emerald-400">开始登录</button></div></div></section>}
        </main>
      </div>
      {loginOpen ? <LoginDialog onClose={() => setLoginOpen(false)} /> : null}
    </div>
  );
}

"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";

export type Locale = "zh" | "en";
export type ThemeMode = "dark" | "light";

export type MessageKey =
  | "nav.overview"
  | "nav.domains"
  | "nav.configs"
  | "nav.experiments"
  | "brand.console"
  | "controls.lang"
  | "controls.theme"
  | "controls.themeDark"
  | "controls.themeLight"
  | "overview.title"
  | "overview.subtitle"
  | "overview.totalConnections"
  | "overview.activeDomains"
  | "overview.warningDomains"
  | "overview.totalMappings"
  | "overview.trend"
  | "overview.lastHour"
  | "overview.runtimeSummary"
  | "overview.criticalAlerts"
  | "overview.latestChange"
  | "overview.refresh"
  | "domains.title"
  | "domains.subtitle"
  | "domains.search"
  | "domains.filterAll"
  | "domains.filterOnline"
  | "domains.filterWarning"
  | "domains.filterOffline"
  | "domains.filterPort"
  | "domains.empty"
  | "domains.paginationSummary"
  | "domains.prev"
  | "domains.next"
  | "domain.lastHeartbeat"
  | "domain.currentConnections"
  | "domain.peakConnections"
  | "domain.mappingCount"
  | "domain.status.online"
  | "domain.status.warning"
  | "domain.status.offline"
  | "common.loading"
  | "common.retry"
  | "configs.title"
  | "configs.subtitle"
  | "configs.core"
  | "configs.frontend"
  | "configs.backend"
  | "configs.mode"
  | "configs.listener"
  | "configs.logLevel"
  | "configs.maxConnections"
  | "configs.readTimeout"
  | "configs.writeTimeout"
  | "configs.upstream"
  | "configs.gzip"
  | "configs.websocket"
  | "configs.cors"
  | "configs.errorPage"
  | "configs.serviceName"
  | "configs.target"
  | "configs.healthCheck"
  | "configs.status"
  | "configs.weight"
  | "experiments.title"
  | "experiments.subtitle"
  | "experiments.risk"
  | "experiments.enabled"
  | "experiments.disabled";

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  theme: ThemeMode;
  setTheme: (theme: ThemeMode) => void;
  t: (key: MessageKey, values?: Record<string, string | number>) => string;
}

const messages: Record<Locale, Record<MessageKey, string>> = {
  zh: {
    "nav.overview": "总览",
    "nav.domains": "域名连接",
    "nav.configs": "网关配置",
    "nav.experiments": "实验特性",
    "brand.console": "网关控制台",
    "controls.lang": "中/EN",
    "controls.theme": "主题",
    "controls.themeDark": "暗黑",
    "controls.themeLight": "明亮",
    "overview.title": "网关总览",
    "overview.subtitle": "网关健康与流量概览",
    "overview.totalConnections": "总连接数",
    "overview.activeDomains": "活动域名",
    "overview.warningDomains": "异常域名",
    "overview.totalMappings": "映射条目",
    "overview.trend": "连接趋势",
    "overview.lastHour": "近 1 小时采样",
    "overview.runtimeSummary": "运行摘要",
    "overview.criticalAlerts": "关键告警",
    "overview.latestChange": "最近配置变更",
    "overview.refresh": "刷新总览数据",
    "domains.title": "域名连接与映射",
    "domains.subtitle": "域名连接状态与端口映射",
    "domains.search": "搜索域名或后端目标",
    "domains.filterAll": "全部状态",
    "domains.filterOnline": "在线",
    "domains.filterWarning": "告警",
    "domains.filterOffline": "离线",
    "domains.filterPort": "按前端端口筛选",
    "domains.empty": "没有匹配条件的域名连接数据。",
    "domains.paginationSummary": "第 {page} / {total} 页，共 {count} 条",
    "domains.prev": "上一页",
    "domains.next": "下一页",
    "domain.lastHeartbeat": "最近心跳",
    "domain.currentConnections": "当前连接",
    "domain.peakConnections": "峰值连接",
    "domain.mappingCount": "映射数量",
    "domain.status.online": "在线",
    "domain.status.warning": "告警",
    "domain.status.offline": "离线",
    "common.loading": "正在加载网关数据...",
    "common.retry": "重新加载",
    "configs.title": "网关配置中心",
    "configs.subtitle": "核心网关 / 前端代理 / 后端服务",
    "configs.core": "核心网关配置",
    "configs.frontend": "前端代理配置",
    "configs.backend": "后端服务配置",
    "configs.mode": "运行模式",
    "configs.listener": "监听地址",
    "configs.logLevel": "日志级别",
    "configs.maxConnections": "最大连接数",
    "configs.readTimeout": "读取超时(ms)",
    "configs.writeTimeout": "写入超时(ms)",
    "configs.upstream": "上游协议",
    "configs.gzip": "Gzip 压缩",
    "configs.websocket": "WebSocket 透传",
    "configs.cors": "CORS 支持",
    "configs.errorPage": "错误页路径",
    "configs.serviceName": "服务名",
    "configs.target": "目标地址",
    "configs.healthCheck": "健康检查",
    "configs.status": "状态",
    "configs.weight": "权重",
    "experiments.title": "实验特性",
    "experiments.subtitle": "AnyTLS 与 Trojan 实验配置",
    "experiments.risk": "风险",
    "experiments.enabled": "已启用（点击关闭）",
    "experiments.disabled": "已关闭（点击启用）",
  },
  en: {
    "nav.overview": "Overview",
    "nav.domains": "Domains",
    "nav.configs": "Configurations",
    "nav.experiments": "Experiments",
    "brand.console": "Gateway Console",
    "controls.lang": "EN/中",
    "controls.theme": "Theme",
    "controls.themeDark": "Dark",
    "controls.themeLight": "Light",
    "overview.title": "Gateway Overview",
    "overview.subtitle": "Gateway Health & Traffic Overview",
    "overview.totalConnections": "Total Connections",
    "overview.activeDomains": "Active Domains",
    "overview.warningDomains": "Warning Domains",
    "overview.totalMappings": "Total Mappings",
    "overview.trend": "Connection Trend",
    "overview.lastHour": "Last 1 Hour",
    "overview.runtimeSummary": "Runtime Summary",
    "overview.criticalAlerts": "Critical Alerts",
    "overview.latestChange": "Latest Change",
    "overview.refresh": "Refresh Overview",
    "domains.title": "Domain Connections & Mappings",
    "domains.subtitle": "Domain health status and port mappings",
    "domains.search": "Search domain or backend target",
    "domains.filterAll": "All Status",
    "domains.filterOnline": "Online",
    "domains.filterWarning": "Warning",
    "domains.filterOffline": "Offline",
    "domains.filterPort": "Filter by frontend port",
    "domains.empty": "No domain data matched current filters.",
    "domains.paginationSummary": "Page {page} / {total}, {count} items",
    "domains.prev": "Previous",
    "domains.next": "Next",
    "domain.lastHeartbeat": "Last heartbeat",
    "domain.currentConnections": "Current",
    "domain.peakConnections": "Peak",
    "domain.mappingCount": "Mappings",
    "domain.status.online": "Online",
    "domain.status.warning": "Warning",
    "domain.status.offline": "Offline",
    "common.loading": "Loading gateway data...",
    "common.retry": "Reload",
    "configs.title": "Gateway Config Center",
    "configs.subtitle": "Core Gateway / Frontend Proxy / Backend Services",
    "configs.core": "Core Gateway Config",
    "configs.frontend": "Frontend Proxy Config",
    "configs.backend": "Backend Services Config",
    "configs.mode": "Mode",
    "configs.listener": "Listener",
    "configs.logLevel": "Log Level",
    "configs.maxConnections": "Max Connections",
    "configs.readTimeout": "Read Timeout(ms)",
    "configs.writeTimeout": "Write Timeout(ms)",
    "configs.upstream": "Upstream Scheme",
    "configs.gzip": "Gzip",
    "configs.websocket": "WebSocket",
    "configs.cors": "CORS",
    "configs.errorPage": "Error Page",
    "configs.serviceName": "Service",
    "configs.target": "Target",
    "configs.healthCheck": "Health Check",
    "configs.status": "Status",
    "configs.weight": "Weight",
    "experiments.title": "Experimental Features",
    "experiments.subtitle": "AnyTLS and Trojan experimental configuration",
    "experiments.risk": "Risk",
    "experiments.enabled": "Enabled (Click to Disable)",
    "experiments.disabled": "Disabled (Click to Enable)",
  },
};

const I18nContext = createContext<I18nContextValue | null>(null);

const LOCALE_KEY = "dashboard.locale";
const THEME_KEY = "dashboard.theme";

function applyTheme(theme: ThemeMode): void {
  document.documentElement.classList.remove("theme-dark", "theme-light");
  document.documentElement.classList.add(theme === "dark" ? "theme-dark" : "theme-light");
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
}

function resolveInitialTheme(): ThemeMode {
  if (typeof window === "undefined") {
    return "dark";
  }
  const savedTheme = window.localStorage.getItem(THEME_KEY);
  if (savedTheme === "dark" || savedTheme === "light") {
    return savedTheme;
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function PreferencesProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocale] = useState<Locale>("zh");
  const [theme, setTheme] = useState<ThemeMode>("dark");

  useEffect(() => {
    const savedLocale = window.localStorage.getItem(LOCALE_KEY);
    if (savedLocale === "zh" || savedLocale === "en") {
      setLocale(savedLocale);
    }
    const initialTheme = resolveInitialTheme();
    setTheme(initialTheme);
    applyTheme(initialTheme);
  }, []);

  const value = useMemo<I18nContextValue>(
    () => ({
      locale,
      setLocale: (nextLocale) => {
        setLocale(nextLocale);
        window.localStorage.setItem(LOCALE_KEY, nextLocale);
      },
      theme,
      setTheme: (nextTheme) => {
        setTheme(nextTheme);
        window.localStorage.setItem(THEME_KEY, nextTheme);
        applyTheme(nextTheme);
      },
      t: (key, values) => {
        let text = messages[locale][key] ?? key;
        if (!values) {
          return text;
        }
        Object.entries(values).forEach(([name, value]) => {
          text = text.replaceAll(`{${name}}`, String(value));
        });
        return text;
      },
    }),
    [locale, theme],
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function usePreferences(): I18nContextValue {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("usePreferences must be used within PreferencesProvider");
  }
  return context;
}

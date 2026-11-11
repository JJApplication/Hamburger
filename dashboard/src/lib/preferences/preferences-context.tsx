"use client";
/* eslint-disable react-hooks/set-state-in-effect */

import { createContext, useContext, useEffect, useMemo, useState } from "react";

export type Locale = "zh" | "en";
export type ThemeMode = "dark" | "light";

export type MessageKey =
  | "nav.overview"
  | "nav.domains"
  | "nav.domainConfig"
  | "nav.services"
  | "nav.staticConfig"
  | "nav.certificates"
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
  | "domains.filterUnknown"
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
  | "domain.status.unknown"
  | "domain.probe"
  | "domain.probing"
  | "domain.probeDone"
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
  | "experiments.disabled"
  | "domainConfig.title"
  | "domainConfig.subtitle"
  | "domainConfig.source"
  | "domainConfig.runtimeMap"
  | "domainConfig.mappings"
  | "domainConfig.services"
  | "domainConfig.search"
  | "domainConfig.filterAll"
  | "domainConfig.frontend"
  | "domainConfig.backend"
  | "domainConfig.custom"
  | "domainConfig.domain"
  | "domainConfig.service"
  | "domainConfig.type"
  | "domainConfig.empty"
  | "domainConfig.unbound"
  | "domainConfig.undefined"
  | "domainConfig.proxy"
  | "domainConfig.static"
  | "domainConfig.forward"
  | "domainConfig.rewrite"
  | "domainConfig.noProxy"
  | "services.title"
  | "services.subtitle"
  | "services.runtimeMap"
  | "services.total"
  | "services.listen"
  | "services.precheck"
  | "services.empty"
  | "staticConfig.title"
  | "staticConfig.subtitle"
  | "staticConfig.runtimeFile"
  | "staticConfig.sites"
  | "staticConfig.aliasSites"
  | "staticConfig.backendSites"
  | "staticConfig.cache"
  | "staticConfig.listen"
  | "staticConfig.balancer"
  | "staticConfig.cacheDir"
  | "staticConfig.memory"
  | "staticConfig.memoryDetail"
  | "staticConfig.expire"
  | "staticConfig.neverExpire"
  | "staticConfig.matchers"
  | "staticConfig.fastConnect"
  | "staticConfig.http3"
  | "staticConfig.on"
  | "staticConfig.off"
  | "staticConfig.headers"
  | "staticConfig.search"
  | "staticConfig.empty"
  | "staticConfig.index"
  | "staticConfig.tryFile"
  | "staticConfig.alias"
  | "staticConfig.forward"
  | "staticConfig.filterAll"
  | "staticConfig.filterAlias"
  | "staticConfig.filterBackend"
  | "staticConfig.access"
  | "staticConfig.compress"
  | "certificates.title"
  | "certificates.subtitle"
  | "certificates.checkedAt"
  | "certificates.total"
  | "certificates.valid"
  | "certificates.expiring"
  | "certificates.problem"
  | "certificates.empty"
  | "certificates.search"
  | "certificates.noMatch"
  | "certificates.domains"
  | "certificates.covered"
  | "certificates.uncovered"
  | "certificates.certFile"
  | "certificates.keyFile"
  | "certificates.keyMatched"
  | "certificates.keyMismatch"
  | "certificates.keyMissing"
  | "certificates.keyUnread"
  | "certificates.fileMissing"
  | "certificates.invalid"
  | "certificates.expired"
  | "certificates.expiringSoon"
  | "certificates.ok"
  | "certificates.keyProblem"
  | "certificates.subject"
  | "certificates.issuer"
  | "certificates.sans"
  | "certificates.validFrom"
  | "certificates.validTo"
  | "certificates.daysLeft"
  | "certificates.daysAgo"
  | "certificates.serial"
  | "certificates.fingerprint"
  | "certificates.chain"
  | "certificates.leaf"
  | "certificates.intermediate"
  | "certificates.minVersion"
  | "certificates.autoTlsOn"
  | "certificates.autoTlsOff"
  | "certificates.noSan"
  | "editor.add"
  | "editor.edit"
  | "editor.delete"
  | "editor.save"
  | "editor.discard"
  | "editor.saving"
  | "editor.reloadHint"
  | "editor.close"
  | "editor.name"
  | "editor.domain"
  | "editor.host"
  | "editor.port"
  | "editor.precheck"
  | "editor.rules"
  | "editor.addRule"
  | "editor.api"
  | "editor.target"
  | "editor.targetService"
  | "editor.targetStatic"
  | "editor.targetHost"
  | "editor.proxyHost"
  | "editor.proxyPort"
  | "editor.useRewrite"
  | "editor.rewrite"
  | "editor.addMapping"
  | "editor.addService"
  | "editor.addSite"
  | "editor.siteType"
  | "editor.root"
  | "editor.actions"
  | "editor.headerName"
  | "editor.headerValue"
  | "editor.addHeader"
  | "editor.addMatcher"
  | "editor.remove"
  | "editor.conflict"
  | "editor.auth"
  | "editor.parse"
  | "editor.emptyName"
  | "editor.duplicateService"
  | "editor.duplicateDomain"
  | "editor.emptyDomain"
  | "editor.emptyBinding"
  | "editor.invalidPort"
  | "editor.emptySite"
  | "editor.duplicateSite"
  | "editor.invalidRule"
  | "editor.failed"
  | "editor.confirmDelete"
  | "editor.cancel"
  | "editor.findService";

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
    "nav.domainConfig": "域名配置",
    "nav.services": "服务配置",
    "nav.staticConfig": "静态配置",
    "nav.certificates": "证书检查",
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
    "domains.filterUnknown": "未探测",
    "domains.filterPort": "按前端端口筛选",
    "domains.empty": "没有匹配条件的域名连接数据。",
    "domains.paginationSummary": "第 {page} / {total} 页，共 {count} 条",
    "domains.prev": "上一页",
    "domains.next": "下一页",
    "domain.lastHeartbeat": "最近探测",
    "domain.currentConnections": "当前连接",
    "domain.peakConnections": "峰值连接",
    "domain.mappingCount": "映射数量",
    "domain.status.online": "在线",
    "domain.status.warning": "告警",
    "domain.status.offline": "离线",
    "domain.status.unknown": "未知",
    "domain.probe": "探测",
    "domain.probing": "探测中",
    "domain.probeDone": "探测完成",
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
    "domainConfig.title": "域名配置",
    "domainConfig.subtitle": "域名映射与服务定义",
    "domainConfig.source": "配置来源",
    "domainConfig.runtimeMap": "网关当前加载的域名映射是 {file}",
    "domainConfig.mappings": "域名映射",
    "domainConfig.services": "服务定义",
    "domainConfig.search": "搜索域名、服务或转发路径",
    "domainConfig.filterAll": "全部类型",
    "domainConfig.frontend": "前端",
    "domainConfig.backend": "后端",
    "domainConfig.custom": "自定义",
    "domainConfig.domain": "域名",
    "domainConfig.service": "服务",
    "domainConfig.type": "类型",
    "domainConfig.empty": "没有匹配的域名配置。",
    "domainConfig.unbound": "未绑定域名",
    "domainConfig.undefined": "未定义",
    "domainConfig.proxy": "转发",
    "domainConfig.static": "静态目录",
    "domainConfig.forward": "转发到",
    "domainConfig.rewrite": "重写",
    "domainConfig.noProxy": "未配置目标",
    "services.title": "服务配置",
    "services.subtitle": "网关加载的服务定义",
    "services.runtimeMap": "主配置的 domain_map 指向 {file}",
    "services.total": "服务",
    "services.listen": "监听",
    "services.precheck": "前置检查",
    "services.empty": "没有匹配的服务。",
    "staticConfig.title": "静态配置",
    "staticConfig.subtitle": "前端站点、缓存与监听",
    "staticConfig.runtimeFile": "主配置的 pxy_frontend_file 指向 {file}",
    "staticConfig.sites": "站点",
    "staticConfig.aliasSites": "别名",
    "staticConfig.backendSites": "转发",
    "staticConfig.cache": "缓存",
    "staticConfig.listen": "监听",
    "staticConfig.balancer": "均衡",
    "staticConfig.cacheDir": "目录",
    "staticConfig.memory": "内存",
    "staticConfig.memoryDetail": "{max} MB · 单文件 {file} MB · {entries} 条 · {seconds} 秒",
    "staticConfig.expire": "过期",
    "staticConfig.neverExpire": "不过期",
    "staticConfig.matchers": "匹配",
    "staticConfig.fastConnect": "快速连接",
    "staticConfig.http3": "HTTP/3",
    "staticConfig.on": "开启",
    "staticConfig.off": "关闭",
    "staticConfig.headers": "响应头",
    "staticConfig.search": "搜索站点、目录或路径",
    "staticConfig.empty": "没有匹配的站点。",
    "staticConfig.index": "入口",
    "staticConfig.tryFile": "回退",
    "staticConfig.alias": "别名",
    "staticConfig.forward": "转发",
    "staticConfig.filterAll": "全部站点",
    "staticConfig.filterAlias": "有别名",
    "staticConfig.filterBackend": "有转发",
    "staticConfig.access": "目录浏览",
    "staticConfig.compress": "压缩",
    "certificates.title": "证书检查",
    "certificates.subtitle": "读取主配置里的 SSL 证书",
    "certificates.checkedAt": "检查时间 {time}",
    "certificates.total": "证书",
    "certificates.valid": "有效",
    "certificates.expiring": "即将到期",
    "certificates.problem": "异常",
    "certificates.empty": "配置里没有证书路径。",
    "certificates.search": "搜索域名或证书",
    "certificates.noMatch": "没有匹配的证书。",
    "certificates.domains": "配置域名",
    "certificates.covered": "已覆盖",
    "certificates.uncovered": "未覆盖",
    "certificates.certFile": "证书",
    "certificates.keyFile": "私钥",
    "certificates.keyMatched": "私钥匹配",
    "certificates.keyMismatch": "私钥不匹配",
    "certificates.keyMissing": "私钥不存在",
    "certificates.keyUnread": "私钥无法读取",
    "certificates.fileMissing": "证书文件不存在",
    "certificates.invalid": "无法解析",
    "certificates.expired": "已过期",
    "certificates.expiringSoon": "即将到期",
    "certificates.ok": "有效",
    "certificates.keyProblem": "私钥异常",
    "certificates.subject": "颁发给",
    "certificates.issuer": "颁发者",
    "certificates.sans": "备用名称",
    "certificates.validFrom": "生效",
    "certificates.validTo": "到期",
    "certificates.daysLeft": "剩余 {days} 天",
    "certificates.daysAgo": "已过期 {days} 天",
    "certificates.serial": "序列号",
    "certificates.fingerprint": "指纹",
    "certificates.chain": "证书链",
    "certificates.leaf": "叶子证书",
    "certificates.intermediate": "中间证书",
    "certificates.minVersion": "最低版本",
    "certificates.autoTlsOn": "自动证书开启",
    "certificates.autoTlsOff": "自动证书关闭",
    "certificates.noSan": "证书未声明备用名称",
    "editor.add": "添加",
    "editor.edit": "编辑",
    "editor.delete": "删除",
    "editor.save": "保存",
    "editor.discard": "放弃",
    "editor.saving": "正在保存",
    "editor.reloadHint": "保存后写入配置文件，网关重载后生效。",
    "editor.close": "关闭",
    "editor.name": "名称",
    "editor.domain": "域名",
    "editor.host": "监听地址",
    "editor.port": "端口",
    "editor.precheck": "前置检查",
    "editor.rules": "转发规则",
    "editor.addRule": "添加规则",
    "editor.api": "路径",
    "editor.target": "目标",
    "editor.targetService": "转到服务",
    "editor.targetStatic": "静态目录",
    "editor.targetHost": "指定主机",
    "editor.proxyHost": "主机",
    "editor.proxyPort": "主机端口",
    "editor.useRewrite": "重写",
    "editor.rewrite": "重写规则",
    "editor.addMapping": "添加映射",
    "editor.addService": "添加服务",
    "editor.addSite": "添加站点",
    "editor.siteType": "站点类型",
    "editor.root": "根目录",
    "editor.actions": "操作",
    "editor.headerName": "头名称",
    "editor.headerValue": "头内容",
    "editor.addHeader": "添加响应头",
    "editor.addMatcher": "添加匹配",
    "editor.remove": "移除",
    "editor.conflict": "配置已在其他地方修改，请刷新后再保存。",
    "editor.auth": "请先登录后再保存。",
    "editor.parse": "写回的配置无法解析。",
    "editor.emptyName": "服务名不能为空。",
    "editor.duplicateService": "服务名重复：{name}",
    "editor.duplicateDomain": "域名重复：{name}",
    "editor.emptyDomain": "域名不能为空。",
    "editor.emptyBinding": "映射必须选择服务。",
    "editor.invalidPort": "{name} 的端口必须是数字。",
    "editor.emptySite": "站点名不能为空。",
    "editor.duplicateSite": "站点名重复：{name}",
    "editor.invalidRule": "{name} 的转发规则缺少目标。",
    "editor.failed": "保存失败：{message}",
    "editor.confirmDelete": "删除 {name}？尚未保存的其他修改会保留。",
    "editor.cancel": "取消",
    "editor.findService": "搜索服务",
  },
  en: {
    "nav.overview": "Overview",
    "nav.domains": "Domains",
    "nav.domainConfig": "Domain Config",
    "nav.services": "Services",
    "nav.staticConfig": "Static Sites",
    "nav.certificates": "Certificates",
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
    "domains.filterUnknown": "Not probed",
    "domains.filterPort": "Filter by frontend port",
    "domains.empty": "No domain data matched current filters.",
    "domains.paginationSummary": "Page {page} / {total}, {count} items",
    "domains.prev": "Previous",
    "domains.next": "Next",
    "domain.lastHeartbeat": "Last probe",
    "domain.currentConnections": "Current",
    "domain.peakConnections": "Peak",
    "domain.mappingCount": "Mappings",
    "domain.status.online": "Online",
    "domain.status.warning": "Warning",
    "domain.status.offline": "Offline",
    "domain.status.unknown": "Unknown",
    "domain.probe": "Probe",
    "domain.probing": "Probing",
    "domain.probeDone": "Probe finished",
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
    "domainConfig.title": "Domain Configuration",
    "domainConfig.subtitle": "Domain mappings and service definitions",
    "domainConfig.source": "Source",
    "domainConfig.runtimeMap": "The gateway currently loads {file}",
    "domainConfig.mappings": "Domain mappings",
    "domainConfig.services": "Service definitions",
    "domainConfig.search": "Search domain, service, or route",
    "domainConfig.filterAll": "All types",
    "domainConfig.frontend": "Frontend",
    "domainConfig.backend": "Backend",
    "domainConfig.custom": "Custom",
    "domainConfig.domain": "Domain",
    "domainConfig.service": "Service",
    "domainConfig.type": "Type",
    "domainConfig.empty": "No domain configuration matched.",
    "domainConfig.unbound": "No domain bound",
    "domainConfig.undefined": "Undefined",
    "domainConfig.proxy": "Route",
    "domainConfig.static": "Static root",
    "domainConfig.forward": "Forward to",
    "domainConfig.rewrite": "Rewrite",
    "domainConfig.noProxy": "No target",
    "services.title": "Service Configuration",
    "services.subtitle": "Services loaded by the gateway",
    "services.runtimeMap": "The main config domain_map points to {file}",
    "services.total": "Services",
    "services.listen": "Listen",
    "services.precheck": "Pre-check",
    "services.empty": "No service matched.",
    "staticConfig.title": "Static Configuration",
    "staticConfig.subtitle": "Frontend sites, cache, and listener",
    "staticConfig.runtimeFile": "The main config pxy_frontend_file points to {file}",
    "staticConfig.sites": "Sites",
    "staticConfig.aliasSites": "Aliases",
    "staticConfig.backendSites": "Proxied",
    "staticConfig.cache": "Cache",
    "staticConfig.listen": "Listener",
    "staticConfig.balancer": "Balancer",
    "staticConfig.cacheDir": "Directory",
    "staticConfig.memory": "Memory",
    "staticConfig.memoryDetail": "{max} MB · {file} MB per file · {entries} entries · {seconds}s",
    "staticConfig.expire": "Expiry",
    "staticConfig.neverExpire": "No expiry",
    "staticConfig.matchers": "Matchers",
    "staticConfig.fastConnect": "Fast connect",
    "staticConfig.http3": "HTTP/3",
    "staticConfig.on": "On",
    "staticConfig.off": "Off",
    "staticConfig.headers": "Headers",
    "staticConfig.search": "Search site, directory, or path",
    "staticConfig.empty": "No site matched.",
    "staticConfig.index": "Index",
    "staticConfig.tryFile": "Fallback",
    "staticConfig.alias": "Alias",
    "staticConfig.forward": "Forward",
    "staticConfig.filterAll": "All sites",
    "staticConfig.filterAlias": "With alias",
    "staticConfig.filterBackend": "With proxy",
    "staticConfig.access": "Directory listing",
    "staticConfig.compress": "Compression",
    "certificates.title": "Certificate Check",
    "certificates.subtitle": "SSL certificates referenced by the main config",
    "certificates.checkedAt": "Checked at {time}",
    "certificates.total": "Certificates",
    "certificates.valid": "Valid",
    "certificates.expiring": "Expiring",
    "certificates.problem": "Problems",
    "certificates.empty": "No certificate path is configured.",
    "certificates.search": "Search a domain or certificate",
    "certificates.noMatch": "No certificate matched.",
    "certificates.domains": "Configured domains",
    "certificates.covered": "Covered",
    "certificates.uncovered": "Not covered",
    "certificates.certFile": "Certificate",
    "certificates.keyFile": "Private key",
    "certificates.keyMatched": "Key matches",
    "certificates.keyMismatch": "Key does not match",
    "certificates.keyMissing": "Key file is missing",
    "certificates.keyUnread": "Key cannot be read",
    "certificates.fileMissing": "Certificate file is missing",
    "certificates.invalid": "Unreadable",
    "certificates.expired": "Expired",
    "certificates.expiringSoon": "Expiring soon",
    "certificates.ok": "Valid",
    "certificates.keyProblem": "Key problem",
    "certificates.subject": "Subject",
    "certificates.issuer": "Issuer",
    "certificates.sans": "Subject alternative names",
    "certificates.validFrom": "Not before",
    "certificates.validTo": "Not after",
    "certificates.daysLeft": "{days} days left",
    "certificates.daysAgo": "Expired {days} days ago",
    "certificates.serial": "Serial",
    "certificates.fingerprint": "Fingerprint",
    "certificates.chain": "Chain",
    "certificates.leaf": "Leaf",
    "certificates.intermediate": "Intermediate",
    "certificates.minVersion": "Minimum version",
    "certificates.autoTlsOn": "Automatic certificates on",
    "certificates.autoTlsOff": "Automatic certificates off",
    "certificates.noSan": "No subject alternative name",
    "editor.add": "Add",
    "editor.edit": "Edit",
    "editor.delete": "Delete",
    "editor.save": "Save",
    "editor.discard": "Discard",
    "editor.saving": "Saving",
    "editor.reloadHint": "Saving writes the file. Reload the gateway to apply it.",
    "editor.close": "Close",
    "editor.name": "Name",
    "editor.domain": "Domain",
    "editor.host": "Listen address",
    "editor.port": "Port",
    "editor.precheck": "Pre-check",
    "editor.rules": "Routes",
    "editor.addRule": "Add route",
    "editor.api": "Path",
    "editor.target": "Target",
    "editor.targetService": "Service",
    "editor.targetStatic": "Static root",
    "editor.targetHost": "Host",
    "editor.proxyHost": "Host",
    "editor.proxyPort": "Host port",
    "editor.useRewrite": "Rewrite",
    "editor.rewrite": "Rewrite rule",
    "editor.addMapping": "Add mapping",
    "editor.addService": "Add service",
    "editor.addSite": "Add site",
    "editor.siteType": "Site type",
    "editor.root": "Root",
    "editor.actions": "Actions",
    "editor.headerName": "Header name",
    "editor.headerValue": "Header value",
    "editor.addHeader": "Add header",
    "editor.addMatcher": "Add matcher",
    "editor.remove": "Remove",
    "editor.conflict": "The file changed elsewhere. Refresh before saving.",
    "editor.auth": "Sign in before saving.",
    "editor.parse": "The written configuration could not be parsed.",
    "editor.emptyName": "Service name is required.",
    "editor.duplicateService": "Duplicate service name: {name}",
    "editor.duplicateDomain": "Duplicate domain: {name}",
    "editor.emptyDomain": "Domain is required.",
    "editor.emptyBinding": "Each mapping needs a service.",
    "editor.invalidPort": "Port for {name} must be a number.",
    "editor.emptySite": "Site name is required.",
    "editor.duplicateSite": "Duplicate site name: {name}",
    "editor.invalidRule": "A route on {name} is missing its target.",
    "editor.failed": "Save failed: {message}",
    "editor.confirmDelete": "Delete {name}? Other unsaved edits stay in the draft.",
    "editor.cancel": "Cancel",
    "editor.findService": "Search services",
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

"use client";

import { FormEvent, useState } from "react";
import { LockKeyhole, X } from "lucide-react";
import { useAuth } from "@/lib/auth/auth-context";

export function LoginDialog({ onClose }: { onClose: () => void }) {
  const { login } = useAuth();
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault(); setPending(true); setError("");
    try { await login(username, password); onClose(); } catch (reason) { setError(reason instanceof Error ? reason.message : "登录失败"); } finally { setPending(false); }
  };
  return <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/65 p-4 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label="登录">
    <form onSubmit={submit} className="panel w-full max-w-sm space-y-4 rounded-2xl border p-5 shadow-2xl">
      <div className="flex items-center justify-between"><div><p className="text-secondary text-xs uppercase tracking-[0.2em]">Hamburger</p><h2 className="mt-1 text-xl font-semibold">登录管理面板</h2></div><button type="button" onClick={onClose} className="text-secondary rounded-full p-2 hover:opacity-75" aria-label="关闭"><X size={18} /></button></div>
      <label className="block text-sm"><span className="text-secondary">用户名</span><input autoComplete="username" value={username} onChange={(event) => setUsername(event.target.value)} className="panel-strong text-primary mt-1 w-full rounded-xl border border-[var(--border)] px-3 py-2 outline-none focus:border-emerald-400" /></label>
      <label className="block text-sm"><span className="text-secondary">密码</span><input autoComplete="current-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} className="panel-strong text-primary mt-1 w-full rounded-xl border border-[var(--border)] px-3 py-2 outline-none focus:border-emerald-400" /></label>
      {error ? <p className="rounded-lg bg-rose-500/10 px-3 py-2 text-sm text-rose-300">{error}</p> : null}
      <button disabled={pending || !username || !password} className="flex w-full items-center justify-center gap-2 rounded-xl bg-emerald-500 px-3 py-2.5 text-sm font-semibold text-slate-950 transition hover:bg-emerald-400 disabled:cursor-not-allowed disabled:opacity-50"><LockKeyhole size={16} />{pending ? "登录中..." : "登录"}</button>
    </form>
  </div>;
}

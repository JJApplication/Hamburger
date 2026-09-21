"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { BurgerScene } from "@/components/visuals/burger-scene";

export default function LogoPage() {
  return <div className="app-bg flex min-h-screen w-full items-center justify-center overflow-hidden bg-[radial-gradient(circle_at_50%_34%,rgba(16,185,129,0.18),transparent_52%)] p-6"><Link href="/" className="text-secondary absolute left-6 top-6 z-20 flex items-center gap-2 rounded-full border border-[var(--border)] bg-black/10 px-4 py-2 text-sm backdrop-blur-md transition hover:text-primary"><ArrowLeft size={16} />返回主页</Link><div className="w-full max-w-2xl"><BurgerScene /><div className="text-center"><p className="text-secondary text-xs uppercase tracking-[0.32em]">Hamburger Observer</p><h1 className="mt-2 text-3xl font-semibold">Gateway Console</h1></div></div></div>;
}

"use client";

import { ChevronDown } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";

import { usePreferences, type MessageKey } from "@/lib/preferences/preferences-context";

const options = [
  { value: "all", label: "domains.filterAll", dot: "bg-slate-400" },
  { value: "online", label: "domains.filterOnline", dot: "bg-emerald-400" },
  { value: "warning", label: "domains.filterWarning", dot: "bg-amber-400" },
  { value: "offline", label: "domains.filterOffline", dot: "bg-rose-400" },
  { value: "unknown", label: "domains.filterUnknown", dot: "bg-slate-500" },
] as const satisfies ReadonlyArray<{ value: string; label: MessageKey; dot: string }>;

export type DomainStatusFilter = (typeof options)[number]["value"];

export function StatusFilter({ value, onChange }: { value: DomainStatusFilter; onChange: (value: DomainStatusFilter) => void }) {
  const { t } = usePreferences();
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const rootRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const listRef = useRef<HTMLUListElement>(null);
  const listId = useId();
  const current = options.find((item) => item.value === value) ?? options[0];

  useEffect(() => {
    if (!open) return;
    const close = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", close);
    listRef.current?.focus();
    return () => document.removeEventListener("mousedown", close);
  }, [open]);

  const choose = (next: DomainStatusFilter) => {
    onChange(next);
    setOpen(false);
    buttonRef.current?.focus();
  };

  const move = (nextIndex: number) => {
    const bounded = (nextIndex + options.length) % options.length;
    setActiveIndex(bounded);
  };

  const onKeyDown = (event: React.KeyboardEvent) => {
    if (!open && (event.key === "ArrowDown" || event.key === "ArrowUp" || event.key === "Enter" || event.key === " ")) {
      event.preventDefault();
      setActiveIndex(Math.max(0, options.findIndex((item) => item.value === value)));
      setOpen(true);
      return;
    }
    if (!open) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      move(activeIndex + 1);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      move(activeIndex - 1);
    } else if (event.key === "Home") {
      event.preventDefault();
      setActiveIndex(0);
    } else if (event.key === "End") {
      event.preventDefault();
      setActiveIndex(options.length - 1);
    } else if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      choose(options[activeIndex].value);
    } else if (event.key === "Escape" || event.key === "Tab") {
      setOpen(false);
      if (event.key === "Escape") {
        event.preventDefault();
        buttonRef.current?.focus();
      }
    }
  };

  return (
    <div ref={rootRef} className="relative" onKeyDown={onKeyDown}>
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        onClick={() => {
          setActiveIndex(Math.max(0, options.findIndex((item) => item.value === value)));
          setOpen((currentOpen) => !currentOpen);
        }}
        className={`panel-strong text-primary flex w-full items-center justify-between gap-3 rounded-xl border px-3 py-2 text-left text-sm outline-none transition ${open ? "border-cyan-400/60" : "border-[var(--border)] focus-visible:border-cyan-400/60"}`}
      >
        <span className="flex min-w-0 items-center gap-2">
          <span className={`size-1.5 shrink-0 rounded-full ${current.dot}`} />
          <span className="truncate">{t(current.label)}</span>
        </span>
        <ChevronDown size={16} className={`text-secondary shrink-0 transition ${open ? "rotate-180" : ""}`} />
      </button>
      {open ? (
        <ul
          ref={listRef}
          id={listId}
          role="listbox"
          tabIndex={-1}
          aria-label={t("domains.filterAll")}
          aria-activedescendant={`${listId}-${options[activeIndex].value}`}
          className="panel absolute z-30 mt-1.5 w-full rounded-xl border p-1 shadow-[0_16px_40px_rgba(0,0,0,0.35)] outline-none"
        >
          {options.map((option, index) => {
            const selected = option.value === value;
            const active = index === activeIndex;
            return (
              <li key={option.value} id={`${listId}-${option.value}`} role="option" aria-selected={selected}>
                <button
                  type="button"
                  onMouseEnter={() => setActiveIndex(index)}
                  onClick={() => choose(option.value)}
                  className={`flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm transition ${selected ? "bg-emerald-500/15 text-emerald-300 [html.theme-light_&]:text-emerald-700" : active ? "bg-[var(--panel-soft)] text-primary" : "text-secondary hover:bg-[var(--panel-soft)] hover:text-primary"}`}
                >
                  <span className={`size-1.5 shrink-0 rounded-full ${option.dot}`} />
                  <span>{t(option.label)}</span>
                </button>
              </li>
            );
          })}
        </ul>
      ) : null}
    </div>
  );
}

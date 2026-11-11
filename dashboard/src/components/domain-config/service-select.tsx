"use client";

import { ChevronDown } from "lucide-react";
import { useEffect, useId, useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { createPortal } from "react-dom";

export interface ServiceChoice {
  name: string;
  type: string;
}

interface MenuFrame {
  left: number;
  width: number;
  maxHeight: number;
  top: number;
  above: boolean;
}

function typeDot(type: string): string {
  if (type === "frontend") return "bg-emerald-400";
  if (type === "backend") return "bg-sky-400";
  return "bg-amber-400";
}

function typeText(type: string): string {
  if (type === "frontend") return "text-emerald-300 [html.theme-light_&]:text-emerald-700";
  if (type === "backend") return "text-sky-300 [html.theme-light_&]:text-sky-700";
  return "text-amber-300 [html.theme-light_&]:text-amber-700";
}

export function ServiceSelect({
  value,
  options,
  placeholder,
  searchPlaceholder,
  emptyText,
  typeLabel,
  onChange,
}: {
  value: string;
  options: ServiceChoice[];
  placeholder: string;
  searchPlaceholder: string;
  emptyText: string;
  typeLabel: (type: string) => string;
  onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const [frame, setFrame] = useState<MenuFrame | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const listId = useId();
  const selected = options.find((option) => option.name === value);
  const visible = useMemo(() => {
    const keyword = query.trim().toLowerCase();
    return keyword
      ? options.filter((option) => option.name.toLowerCase().includes(keyword) || typeLabel(option.type).toLowerCase().includes(keyword))
      : options;
  }, [options, query, typeLabel]);

  useLayoutEffect(() => {
    if (!open) return;
    const update = () => {
      const button = buttonRef.current;
      if (!button) return;
      const rect = button.getBoundingClientRect();
      const gap = 6;
      const margin = 8;
      const width = Math.min(Math.max(rect.width, 240), window.innerWidth - margin * 2);
      const left = Math.min(Math.max(margin, rect.left), window.innerWidth - width - margin);
      const spaceBelow = window.innerHeight - rect.bottom - gap - margin;
      const spaceAbove = rect.top - gap - margin;
      const above = spaceBelow < 240 && spaceAbove > spaceBelow;
      setFrame({
        left,
        width,
        maxHeight: Math.max(180, Math.min(320, above ? spaceAbove : spaceBelow)),
        top: above ? rect.top - gap : rect.bottom + gap,
        above,
      });
    };
    update();
    const close = (event: MouseEvent) => {
      const target = event.target as Node;
      if (rootRef.current?.contains(target) || menuRef.current?.contains(target)) return;
      setQuery("");
      setOpen(false);
    };
    window.addEventListener("resize", update);
    window.addEventListener("scroll", update, true);
    document.addEventListener("mousedown", close);
    return () => {
      window.removeEventListener("resize", update);
      window.removeEventListener("scroll", update, true);
      document.removeEventListener("mousedown", close);
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const item = document.getElementById(`${listId}-${activeIndex}`);
    const list = item?.closest("ul");
    if (!item || !list) return;
    const top = item.offsetTop;
    const bottom = top + item.offsetHeight;
    if (top < list.scrollTop) list.scrollTop = top;
    else if (bottom > list.scrollTop + list.clientHeight) list.scrollTop = bottom - list.clientHeight;
  }, [open, activeIndex, listId, visible]);

  const choose = (next: string) => {
    onChange(next);
    setOpen(false);
    setQuery("");
    buttonRef.current?.focus();
  };

  const onKeyDown = (event: KeyboardEvent) => {
    if (!open && event.key === "ArrowDown") {
      event.preventDefault();
      setQuery("");
      setActiveIndex(Math.max(0, visible.findIndex((option) => option.name === value)));
      setOpen(true);
      return;
    }
    if (!open) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((index) => Math.min(visible.length - 1, index + 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((index) => Math.max(0, index - 1));
    } else if (event.key === "Enter") {
      event.preventDefault();
      const option = visible[activeIndex];
      if (option) choose(option.name);
    } else if (event.key === "Escape") {
      event.preventDefault();
      setQuery("");
      setOpen(false);
      buttonRef.current?.focus();
    }
  };

  const menu = open && frame ? (
    <div
      ref={menuRef}
      onKeyDown={onKeyDown}
      className="panel fixed z-40 flex flex-col overflow-hidden rounded-xl border shadow-[0_18px_40px_rgba(15,23,42,0.22)]"
      style={{
        left: frame.left,
        width: frame.width,
        maxHeight: frame.maxHeight,
        top: frame.above ? undefined : frame.top,
        bottom: frame.above ? window.innerHeight - frame.top : undefined,
      }}
    >
      <div className="border-b border-[var(--border)] p-1.5">
        <input
          ref={searchRef}
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setActiveIndex(0);
          }}
          placeholder={searchPlaceholder}
          autoFocus
          className="panel-strong text-primary placeholder:text-tertiary w-full rounded-lg border border-[var(--border)] px-2.5 py-1.5 text-sm outline-none focus:border-emerald-400/70"
        />
      </div>
      <ul id={listId} role="listbox" aria-label={placeholder} className="min-h-0 flex-1 overflow-y-auto p-1">
        {visible.length === 0 ? <li className="text-secondary px-2.5 py-2 text-sm">{emptyText}</li> : visible.map((option, index) => {
          const current = option.name === value;
          const active = index === activeIndex;
          return (
            <li key={option.name} id={`${listId}-${index}`} role="option" aria-selected={current}>
              <button
                type="button"
                onMouseEnter={() => setActiveIndex(index)}
                onClick={() => choose(option.name)}
                className={`flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm transition ${current ? "bg-emerald-500/15 text-emerald-300 [html.theme-light_&]:text-emerald-700" : active ? "bg-[var(--panel-soft)] text-primary" : "text-secondary hover:bg-[var(--panel-soft)] hover:text-primary"}`}
              >
                <span className={`size-1.5 shrink-0 rounded-full ${typeDot(option.type)}`} />
                <span className="min-w-0 flex-1 truncate">{option.name}</span>
                <span className={`shrink-0 text-xs ${current ? "" : typeText(option.type)}`}>{typeLabel(option.type)}</span>
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  ) : null;

  return (
    <div ref={rootRef} className="relative min-w-[9.5rem]" onKeyDown={onKeyDown}>
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        onClick={() => {
          setQuery("");
          setActiveIndex(Math.max(0, visible.findIndex((option) => option.name === value)));
          setOpen((current) => !current);
        }}
        className={`panel-strong flex w-full items-center justify-between gap-2 rounded-xl border px-3 py-2 text-left text-sm outline-none transition ${open ? "border-emerald-400/70" : "border-[var(--border)] focus-visible:border-emerald-400/70"}`}
      >
        <span className="flex min-w-0 items-center gap-2">
          <span className={`size-1.5 shrink-0 rounded-full ${selected ? typeDot(selected.type) : "bg-slate-400"}`} />
          <span className={`truncate ${selected ? "text-primary" : "text-tertiary"}`}>{selected?.name || value || placeholder}</span>
        </span>
        <ChevronDown size={15} className={`text-secondary shrink-0 transition ${open ? "rotate-180" : ""}`} />
      </button>
      {typeof document === "undefined" ? null : createPortal(menu, document.body)}
    </div>
  );
}

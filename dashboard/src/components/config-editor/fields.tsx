"use client";

import { ChevronDown } from "lucide-react";
import { useEffect, useId, useLayoutEffect, useRef, useState, type KeyboardEvent } from "react";
import { createPortal } from "react-dom";

export const fieldClass = "panel-strong text-primary placeholder:text-tertiary w-full rounded-xl border border-[var(--border)] px-3 py-2 text-sm outline-none focus:border-emerald-400/70";

interface MenuFrame {
  left: number;
  width: number;
  maxHeight: number;
  top: number;
  above: boolean;
}

export function Field({ label, value, onChange, mono = false }: { label: string; value: string; onChange: (value: string) => void; mono?: boolean }) {
  return (
    <label className="block text-sm">
      <span className="text-secondary">{label}</span>
      <input value={value} onChange={(event) => onChange(event.target.value)} className={`${fieldClass} mt-1${mono ? " font-mono" : ""}`} />
    </label>
  );
}

export function SelectField({ label, value, options, onChange }: { label: string; value: string; options: { value: string; label: string }[]; onChange: (value: string) => void }) {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const [frame, setFrame] = useState<MenuFrame | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const listId = useId();
  const selected = options.find((option) => option.value === value) ?? options.find((option) => option.label === value);

  useLayoutEffect(() => {
    if (!open) return;
    const update = () => {
      const button = buttonRef.current;
      if (!button) return;
      const rect = button.getBoundingClientRect();
      const gap = 6;
      const margin = 8;
      const width = Math.min(rect.width, window.innerWidth - margin * 2);
      const left = Math.min(Math.max(margin, rect.left), window.innerWidth - width - margin);
      const spaceBelow = window.innerHeight - rect.bottom - gap - margin;
      const spaceAbove = rect.top - gap - margin;
      const above = spaceBelow < 160 && spaceAbove > spaceBelow;
      setFrame({
        left,
        width,
        maxHeight: Math.max(120, Math.min(280, above ? spaceAbove : spaceBelow)),
        top: above ? rect.top - gap : rect.bottom + gap,
        above,
      });
    };
    update();
    const close = (event: MouseEvent) => {
      const target = event.target as Node;
      if (rootRef.current?.contains(target) || menuRef.current?.contains(target)) return;
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
  }, [open, activeIndex, listId]);

  const choose = (next: string) => {
    onChange(next);
    setOpen(false);
    buttonRef.current?.focus();
  };

  const onKeyDown = (event: KeyboardEvent) => {
    if (!open && event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex(Math.max(0, options.findIndex((option) => option.value === value)));
      setOpen(true);
      return;
    }
    if (!open) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((index) => Math.min(options.length - 1, index + 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((index) => Math.max(0, index - 1));
    } else if (event.key === "Enter") {
      event.preventDefault();
      const option = options[activeIndex];
      if (option) choose(option.value);
    } else if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      setOpen(false);
    }
  };

  const menu = open && frame ? (
    <div
      ref={menuRef}
      onKeyDown={onKeyDown}
      onMouseDown={(event) => event.stopPropagation()}
      className="panel fixed z-[60] flex flex-col overflow-hidden rounded-xl border shadow-[0_18px_40px_rgba(15,23,42,0.22)]"
      style={{
        left: frame.left,
        width: frame.width,
        maxHeight: frame.maxHeight,
        top: frame.above ? undefined : frame.top,
        bottom: frame.above ? window.innerHeight - frame.top : undefined,
      }}
    >
      <ul id={listId} role="listbox" aria-label={label} className="min-h-0 flex-1 overflow-y-auto p-1">
        {options.map((option, index) => {
          const current = option.value === value;
          const active = index === activeIndex;
          return (
            <li key={option.value} id={`${listId}-${index}`} role="option" aria-selected={current}>
              <button
                type="button"
                onMouseEnter={() => setActiveIndex(index)}
                onClick={() => choose(option.value)}
                className={`flex w-full items-center rounded-lg px-2.5 py-2 text-left text-sm transition ${current ? "bg-emerald-500/15 text-emerald-300 [html.theme-light_&]:text-emerald-700" : active ? "bg-[var(--panel-soft)] text-primary" : "text-secondary hover:bg-[var(--panel-soft)] hover:text-primary"}`}
              >
                <span className="min-w-0 flex-1 truncate">{option.label}</span>
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  ) : null;

  return (
    <div className="block text-sm">
      <span className="text-secondary">{label}</span>
      <div ref={rootRef} className="relative mt-1" onKeyDown={onKeyDown}>
        <button
          ref={buttonRef}
          type="button"
          aria-haspopup="listbox"
          aria-expanded={open}
          aria-controls={listId}
          onClick={() => {
            setActiveIndex(Math.max(0, options.findIndex((option) => option.value === value)));
            setOpen((current) => !current);
          }}
          className={`panel-strong flex w-full items-center justify-between gap-2 rounded-xl border px-3 py-2 text-left text-sm outline-none transition ${open ? "border-emerald-400/70" : "border-[var(--border)] focus-visible:border-emerald-400/70"}`}
        >
          <span className={`truncate ${selected ? "text-primary" : "text-tertiary"}`}>{selected?.label || value}</span>
          <ChevronDown size={15} className={`text-secondary shrink-0 transition ${open ? "rotate-180" : ""}`} />
        </button>
        {typeof document === "undefined" ? null : createPortal(menu, document.body)}
      </div>
    </div>
  );
}

export function CheckField({ label, checked, onChange }: { label: string; checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <label className="panel-soft flex items-center justify-between rounded-xl px-3 py-2 text-sm">
      <span>{label}</span>
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
    </label>
  );
}

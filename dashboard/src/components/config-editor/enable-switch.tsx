export function EnableSwitch({ label, checked, disabled, onChange, hideLabel = false }: { label: string; checked: boolean; disabled: boolean; onChange: (value: boolean) => void; hideLabel?: boolean }) {
  return (
    <label className={`inline-flex shrink-0 items-center gap-2 rounded-full focus-within:ring-2 focus-within:ring-emerald-400/70 ${hideLabel ? "p-0.5" : `border py-1 pr-1 pl-3 text-xs font-medium ${checked ? "border-emerald-400/45 bg-emerald-500/15 text-emerald-200 [html.theme-light_&]:text-emerald-800" : "border-[var(--border)] text-secondary"}`} ${disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer"}`}>
      <span className={hideLabel ? "sr-only" : undefined}>{label}</span>
      <span aria-hidden="true" className={`relative h-5 w-9 rounded-full ${checked ? "bg-emerald-500" : "bg-white/15 [html.theme-light_&]:bg-slate-300"}`}>
        <span className={`absolute top-0.5 left-0.5 size-4 rounded-full bg-white shadow-sm ring-1 ring-slate-900/15 motion-safe:transition-transform ${checked ? "translate-x-4" : ""}`} />
      </span>
      <input type="checkbox" className="sr-only" checked={checked} disabled={disabled} aria-label={label} onChange={(event) => onChange(event.target.checked)} />
    </label>
  );
}

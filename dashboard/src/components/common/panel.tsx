import { clsx } from "clsx";

interface PanelProps {
  className?: string;
  children: React.ReactNode;
}

export function Panel({ className, children }: PanelProps) {
  return <section className={clsx("panel rounded-2xl border p-4", className)}>{children}</section>;
}

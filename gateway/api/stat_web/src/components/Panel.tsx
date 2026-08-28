import type { ReactNode } from "react";

export function Panel({ title, eyebrow, action, children, className = "" }: { title: string; eyebrow?: string; action?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={`panel ${className}`}>
      <div className="panel-heading">
        <div>
          {eyebrow && <div className="panel-eyebrow">{eyebrow}</div>}
          <h2>{title}</h2>
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

export function SectionMessage({ kind = "empty", title, detail }: { kind?: "empty" | "error" | "loading"; title: string; detail?: string }) {
  return <div className={`section-message ${kind}`}><div className="message-mark">{kind === "error" ? "!" : kind === "loading" ? "…" : "○"}</div><div><strong>{title}</strong>{detail && <p>{detail}</p>}</div></div>;
}

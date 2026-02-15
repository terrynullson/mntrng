import type { ReactNode } from "react";

type StateKind = "info" | "error";

type StatePanelProps = {
  kind?: StateKind;
  children: ReactNode;
};

export function StatePanel({ kind = "info", children }: StatePanelProps) {
  const className = kind === "error" ? "state state-error" : "state state-info";
  return <p className={className}>{children}</p>;
}

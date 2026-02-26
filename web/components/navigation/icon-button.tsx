"use client";

import type { ButtonHTMLAttributes } from "react";

type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  label: string;
  tooltip?: string;
  destructive?: boolean;
  children: any;
};

export function IconButton({
  label,
  tooltip,
  destructive = false,
  className = "",
  children,
  ...props
}: IconButtonProps) {
  const styleClass = destructive ? "icon-button icon-button-danger" : "icon-button";

  return (
    <button
      {...props}
      type={props.type ?? "button"}
      className={`${styleClass} ${className}`.trim()}
      aria-label={label}
      title={tooltip ?? label}
    >
      {children as any}
    </button>
  );
}

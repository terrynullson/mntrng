"use client";

import type { ButtonHTMLAttributes, ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "danger" | "ghost";

type AppButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  children: ReactNode;
  isLoading?: boolean;
};

export function AppButton({
  variant = "primary",
  className = "",
  children,
  isLoading = false,
  disabled,
  ...props
}: AppButtonProps) {
  const variantClass =
    variant === "secondary"
      ? "button-secondary"
      : variant === "danger"
        ? "button-danger"
        : variant === "ghost"
          ? "button-ghost"
          : "button-primary";

  const isDisabled = disabled ?? isLoading;

  return (
    <button
      {...props}
      type={props.type ?? "button"}
      disabled={isDisabled}
      data-loading={isLoading ? "true" : undefined}
      className={`${variantClass} ${className}`.trim()}
    >
      <span className="button-label">{children}</span>
    </button>
  );
}

"use client";

import type { HTMLAttributes, ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "danger" | "ghost";

type AppButtonProps = HTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  children: ReactNode;
  isLoading?: boolean;
  type?: "button" | "submit" | "reset";
  disabled?: boolean;
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
      <span className="button-label">{children as any}</span>
    </button>
  );
}

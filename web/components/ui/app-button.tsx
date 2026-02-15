import type { ButtonHTMLAttributes, ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "danger" | "ghost";

type AppButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  children: ReactNode;
};

export function AppButton({
  variant = "primary",
  className = "",
  children,
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

  return (
    <button {...props} className={`${variantClass} ${className}`.trim()}>
      {children}
    </button>
  );
}

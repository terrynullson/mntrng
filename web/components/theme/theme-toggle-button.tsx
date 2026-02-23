"use client";

import { AppButton } from "@/components/ui/app-button";
import { useTheme } from "@/components/theme/theme-provider";

export function ThemeToggleButton() {
  const { theme, toggleTheme } = useTheme();
  return (
    <AppButton
      type="button"
      variant="ghost"
      onClick={toggleTheme}
      aria-label={theme === "light" ? "Switch to dark theme" : "Switch to light theme"}
    >
      {theme === "light" ? "Dark" : "Light"}
    </AppButton>
  );
}

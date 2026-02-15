import type { Metadata } from "next";
import type { ReactNode } from "react";

import { AppRoot } from "@/components/layout/app-root";

import "./globals.css";

export const metadata: Metadata = {
  title: "HLS Monitoring Admin",
  description: "Secure admin interface for HLS platform"
};

type RootLayoutProps = {
  children: ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <html lang="en">
      <body>
        <AppRoot>{children}</AppRoot>
      </body>
    </html>
  );
}

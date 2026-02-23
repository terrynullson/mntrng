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

const THEME_SCRIPT = `(function(){var k='hls-admin-theme';var t=localStorage.getItem(k);if(t!=='light'&&t!=='dark'){t=window.matchMedia&&window.matchMedia('(prefers-color-scheme:dark)').matches?'dark':'light';}if(t)document.documentElement.setAttribute('data-theme',t);})();`;

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_SCRIPT }} />
      </head>
      <body>
        <AppRoot>{children}</AppRoot>
      </body>
    </html>
  );
}

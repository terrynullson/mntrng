import type { Metadata } from "next";

import { AppRoot } from "@/components/layout/app-root";

import "./globals.css";

export const metadata: Metadata = {
  title: "HLS Monitoring Admin",
  description: "Secure admin interface for HLS platform"
};

/** children: any для совместимости Next.js LayoutProps с дублированием react-типов (shim + @types/react). */
type RootLayoutProps = {
  children: any;
};

const THEME_SCRIPT = `(function(){var k='hls-admin-theme';var t=localStorage.getItem(k);if(t!=='light'&&t!=='dark'){t='dark';}if(t)document.documentElement.setAttribute('data-theme',t);})();`;

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

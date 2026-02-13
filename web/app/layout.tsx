import type { Metadata } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import "./globals.css";

export const metadata: Metadata = {
  title: "HLS Monitoring Admin",
  description: "Admin shell for stream monitoring"
};

type RootLayoutProps = {
  children: ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <html lang="en">
      <body>
        <div className="admin-shell">
          <header className="admin-header">
            <div>
              <p className="admin-header-kicker">HLS Monitoring</p>
              <h1 className="admin-header-title">Admin Console</h1>
            </div>
          </header>

          <div className="admin-body">
            <nav className="admin-nav" aria-label="Primary navigation">
              <Link href="/">Overview</Link>
              <Link href="/streams">Streams</Link>
            </nav>

            <main className="admin-content">{children}</main>
          </div>
        </div>
      </body>
    </html>
  );
}

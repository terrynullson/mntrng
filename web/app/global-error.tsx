"use client";

/**
 * Catches runtime errors in the root layout to avoid showing Next.js default error
 * (which can reference next/dist/pages/_app in some environments).
 */
export default function GlobalError({
  error,
  reset
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en">
      <body style={{ margin: 0, fontFamily: "system-ui", padding: 24, background: "#f3f5f8", color: "#0f172a" }}>
        <div style={{ maxWidth: 480, margin: "0 auto" }}>
          <h2 style={{ margin: "0 0 12px", fontSize: "1.25rem" }}>Something went wrong</h2>
          <p style={{ margin: "0 0 16px", color: "#475569", fontSize: 14 }}>
            {error.message || "An unexpected error occurred."}
          </p>
          <button
            type="button"
            onClick={() => reset()}
            style={{
              padding: "10px 16px",
              borderRadius: 10,
              border: "1px solid #0f766e",
              background: "#0f766e",
              color: "#fff",
              fontWeight: 600,
              cursor: "pointer",
              fontSize: 14
            }}
          >
            Try again
          </button>
          <p style={{ margin: "16px 0 0", fontSize: 14 }}>
            <a href="/login" style={{ color: "#0f766e", textDecoration: "underline" }}>
              Back to login
            </a>
          </p>
        </div>
      </body>
    </html>
  );
}

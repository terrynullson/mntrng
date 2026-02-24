/**
 * Fallback type declarations for 'framer-motion' when node_modules is missing or locked.
 */
declare module "framer-motion" {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  export const motion: any;
  /** Returns true if user prefers reduced motion (e.g. prefers-reduced-motion: reduce). */
  export function useReducedMotion(): boolean | null;
  /** Wrapper for exit animations. */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  export const AnimatePresence: any;
}

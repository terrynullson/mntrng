/**
 * Fallback type declarations for 'framer-motion' when node_modules is missing or locked.
 */
declare module "framer-motion" {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  export const motion: any;
}

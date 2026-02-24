/**
 * Fallback type declarations for 'react' when node_modules is missing or locked.
 * After successful npm install, @types/react from node_modules will be used.
 */
declare module "react" {
  export type ReactNode =
    | string
    | number
    | boolean
    | null
    | undefined
    | ReactElement
    | ReactNode[];

  interface ReactElement {
    type: unknown;
    props: unknown;
    key: string | null;
  }

  export interface SyntheticEvent<T = Element> {
    target: T;
    currentTarget: T;
    preventDefault(): void;
    stopPropagation(): void;
  }

  export interface FormEvent<T = Element> extends SyntheticEvent<T> {}

  export interface ChangeEvent<T = Element> extends SyntheticEvent<T> {
    target: T & { value: string; valueAsNumber?: number };
  }

  export interface Context<T> {
    Provider: ComponentType<{ value: T; children?: ReactNode }>;
    Consumer: unknown;
  }

  export function createContext<T>(defaultValue: T | null): Context<T | null>;

  export function useContext<T>(context: Context<T | null>): T | null;

  export function useState<S>(initialState: S): [S, (value: S | ((prev: S) => S)) => void];

  export function useEffect(effect: () => void | (() => void), deps?: unknown[]): void;

  export function useCallback<T>(fn: T, deps: unknown[]): T;

  export function useMemo<T>(factory: () => T, deps: unknown[]): T;

  export interface ComponentType<P = unknown> {
    (props: P): ReactElement | null;
  }
}

declare global {
  namespace JSX {
    interface IntrinsicElements {
      [elemName: string]: Record<string, unknown>;
    }
  }
}

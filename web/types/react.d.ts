/**
 * Fallback type declarations for 'react' when node_modules is missing or locked.
 * After successful npm install, @types/react from node_modules will be used.
 */
declare module "react" {
  /** Совместимость с @types/react: children обязателен, чтобы наш ReactNode был присваиваем к React.ReactNode. */
  export interface ReactPortal {
    key: string | null;
    children: ReactNode;
    type: unknown;
    props: unknown;
  }

  /** Iterable для совместимости с React 18 React.ReactNode. */
  export type ReactNode =
    | string
    | number
    | bigint
    | boolean
    | null
    | undefined
    | ReactElement
    | ReactPortal
    | ReactNode[]
    | Iterable<ReactNode>;

  interface ReactElement {
    type: unknown;
    props: unknown;
    key: string | null;
    /** Опционально для совместимости с созданием элементов без children. */
    children?: ReactNode;
  }

  /** Минимальный набор для совместимости с @types/react при использовании шима. */
  export interface HTMLAttributes<T = unknown> {
    className?: string;
    id?: string;
    style?: Record<string, string | number | undefined>;
    role?: string;
    tabIndex?: number;
    [key: string]: unknown;
  }

  /** Атрибуты кнопки: расширяют HTMLAttributes для type, disabled и т.д. */
  export interface ButtonHTMLAttributes<T = unknown> extends HTMLAttributes<T> {
    disabled?: boolean;
    form?: string;
    formAction?: string;
    formEncType?: string;
    formMethod?: string;
    formNoValidate?: boolean;
    formTarget?: string;
    name?: string;
    type?: "button" | "submit" | "reset";
    value?: string | string[] | number;
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

  export function useRef<T>(initialValue: T | null): { current: T | null };
  export function useRef<T>(initialValue: T): { current: T };

  /** Обёртка для мемоизации компонента. */
  export function memo<P extends object>(
    Component: (props: P) => ReactNode,
    propsAreEqual?: (prevProps: P, nextProps: P) => boolean
  ): (props: P) => ReactNode;

  export type PropsWithChildren<P = unknown> = P & { children?: ReactNode };

  /** Возврат ReactNode для совместимости с JSX и @types/react. */
  export interface ComponentType<P = unknown> {
    (props: P): ReactNode;
  }
}

declare global {
  namespace JSX {
    interface IntrinsicElements {
      [elemName: string]: Record<string, unknown>;
    }
  }
}

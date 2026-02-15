type SkeletonProps = {
  lines?: number;
  className?: string;
};

export function SkeletonBlock({ lines = 3, className = "" }: SkeletonProps) {
  return (
    <div className={`skeleton-block ${className}`.trim()} aria-hidden="true">
      {Array.from({ length: lines }).map((_, index) => (
        <span key={index} className="skeleton-line" />
      ))}
    </div>
  );
}

import { type ReactNode, useEffect } from "react";

interface PageHeaderProps {
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
}

/**
 * The page's single <h1>. It is programmatically focusable so focus moves
 * here after client-side navigation (see useRouteFocus).
 */
export function PageHeader({ title, description, actions }: PageHeaderProps) {
  useEffect(() => {
    document.title = `${title} · OpsGrid`;
  }, [title]);
  return (
    <header className="flex flex-wrap items-start justify-between gap-4 pb-6">
      <div className="min-w-0 space-y-1">
        <h1
          tabIndex={-1}
          className="text-xl font-semibold tracking-tight text-text-primary outline-none"
        >
          {title}
        </h1>
        {description && <p className="text-sm text-text-secondary">{description}</p>}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </header>
  );
}

import type { ReactNode } from "react";
import { useLayout } from "../LayoutContext";
import { Icon } from "./Icon";

type NewsHeaderProps = {
  source: string;
  updated: string;
  action: string;
  running: boolean;
  onRun(): void;
  children: ReactNode;
};

export function NewsHeader({
  source,
  updated,
  action,
  running,
  onRun,
  children,
}: NewsHeaderProps) {
  const { closeNewsPage } = useLayout();

  return (
    <>
      <header className="panel-head flex flex-none items-center gap-2 border-b border-line px-3 py-2">
        <button
          className="btn btn-ghost h-7 w-7 flex-none justify-center px-0"
          title="Back to news (Esc)"
          aria-label="Back to news"
          onClick={closeNewsPage}
        >
          <Icon name="chevron-left" />
        </button>
        <nav
          aria-label="Breadcrumb"
          className="flex min-w-0 items-center gap-2 text-body"
        >
          <button className="text-soft hover:text-ink" onClick={closeNewsPage}>
            News
          </button>
          <span className="text-dim">/</span>
          <h2 className="truncate font-semibold">{source}</h2>
        </nav>
        <span className="mono ml-auto flex-none text-small text-dim">
          {updated}
        </span>
        <button
          className="btn h-7 w-7 flex-none justify-center px-0"
          title={action}
          aria-label={action}
          disabled={running}
          onClick={onRun}
        >
          <span
            className={running ? "inline-flex animate-spin" : "inline-flex"}
          >
            <Icon name="refresh" size={13} />
          </span>
        </button>
      </header>
      <p className="flex-none border-b border-line px-6 py-2.5 text-small text-dim">
        {children}
      </p>
    </>
  );
}

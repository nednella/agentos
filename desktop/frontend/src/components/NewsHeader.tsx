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
  const { closeNewsPage, closeCentre } = useLayout();

  return (
    <>
      <header className="panel-head flex flex-none items-center gap-2 border-b border-line px-4 py-2.5">
        <nav
          aria-label="Breadcrumb"
          className="flex min-w-0 items-center gap-2 text-body"
        >
          <button
            className="inline-flex items-center gap-1 text-soft hover:text-ink"
            title="Back to news (Esc)"
            onClick={closeNewsPage}
          >
            <Icon name="back" size={13} />
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
        <button
          className="btn btn-ghost h-7 w-7 flex-none justify-center px-0"
          title="Close"
          aria-label="Close news"
          onClick={closeCentre}
        >
          <Icon name="close" />
        </button>
      </header>
      <p className="flex-none border-b border-line px-6 py-2.5 text-small text-dim">
        {children}
      </p>
    </>
  );
}

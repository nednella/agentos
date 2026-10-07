import type { CSSProperties } from "react";
import { useAgentos } from "../AgentosContext";
import { Icon } from "./Icon";

type ShellListProps = {
  className?: string;
  style?: CSSProperties;
  onPick?(): void;
};

export function ShellList({
  className = "flex-none border-l border-line bg-surface",
  style,
  onPick,
}: ShellListProps) {
  const {
    shellIds,
    shellId,
    selectShell,
    newShell,
    closeShell,
    report,
    focus,
  } = useAgentos();

  return (
    <div
      className={`flex w-40 min-h-0 flex-col ${className}`}
      style={style}
      role="tablist"
      aria-label="Shells"
      aria-orientation="vertical"
    >
      <button
        className="row h-8 flex-none items-center gap-2 border-b border-line px-3 text-small text-soft"
        onClick={() => {
          report(async () => {
            await newShell();
            focus("shell");
          });
          onPick?.();
        }}
      >
        <Icon name="plus" size={13} />
        New shell
      </button>
      <div className="min-h-0 flex-1 overflow-y-auto">
        {shellIds.map((id, i) => {
          const active = id === shellId;
          return (
            <div
              key={id}
              className="row group h-7 flex-none items-center justify-between pr-1.5 text-small"
              data-selected={active}
              style={{
                boxShadow: active ? "inset 2px 0 0 var(--accent)" : undefined,
                color: active ? "var(--text)" : "var(--text-soft)",
              }}
            >
              <button
                role="tab"
                aria-selected={active}
                className="h-full flex-1 pl-3 text-left"
                onClick={() => {
                  selectShell(id);
                  focus("shell");
                  onPick?.();
                }}
              >
                Shell {i + 1}
              </button>
              {shellIds.length > 1 && (
                <button
                  className="btn btn-ghost h-5 w-5 justify-center px-0 opacity-0 focus-visible:opacity-100 group-hover:opacity-100"
                  title="Close shell"
                  aria-label={`Close shell ${i + 1}`}
                  onClick={() => report(() => closeShell(id))}
                >
                  <Icon name="trash" size={13} />
                </button>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

import { useEffect, useRef } from "react";
import { useAgentos } from "../AgentosContext";
import { ago, useNow } from "../time";
import { NewsHeader } from "./NewsHeader";
import { NewsItemRow } from "./NewsItemRow";

const dayLabel = (date: string) => {
  const [year, month, day] = date.split("-").map(Number);
  return new Date(year, month - 1, day).toLocaleDateString(undefined, {
    weekday: "long",
    month: "short",
    day: "numeric",
  });
};

export function NewsView() {
  const { news, refreshNews, markNewsSeen, report, focusRequest } =
    useAgentos();
  const now = useNow();
  const panel = useRef<HTMLElement>(null);
  const issues = news?.issues ?? [];
  const newest = Math.max(0, ...issues.map((i) => i.fetchedAt));

  useEffect(() => panel.current?.focus(), []);
  useEffect(() => {
    if (focusRequest.target === "terminal") panel.current?.focus();
  }, [focusRequest]);
  useEffect(() => markNewsSeen(), [newest, markNewsSeen]);

  return (
    <section
      ref={panel}
      data-panel="terminal"
      tabIndex={-1}
      aria-label="TLDR Dev"
      className="panel flex h-full min-h-0 min-w-0 flex-col overflow-hidden"
    >
      <NewsHeader
        source="TLDR Dev"
        updated={
          news && news.lastFetchAt > 0
            ? ago(news.lastFetchAt, now)
            : "never fetched"
        }
        action="Refresh"
        running={!news || news.running}
        onRun={() => report(refreshNews)}
      >
        Daily software engineering newsletter · checks every hour · keeps the
        last 7 issues
      </NewsHeader>
      {news?.error && (
        <p className="flex-none border-b border-line px-4 py-2 text-small text-danger">
          {news.error}
        </p>
      )}
      <div className="min-h-0 flex-1 overflow-y-auto">
        {issues.length === 0 && (
          <div className="flex flex-col gap-1 px-6 py-16 text-center">
            <p className="text-body font-medium">No news yet</p>
            <p className="text-small text-dim">
              The latest issues of the TLDR Dev newsletter appear here after the
              first fetch.
            </p>
          </div>
        )}
        <div className="mx-auto max-w-3xl pb-8">
          {issues.map((issue) => (
            <section key={issue.date}>
              <h3 className="sticky top-0 flex flex-col gap-0.5 border-b border-line bg-surface px-6 pt-6 pb-3">
                <span className="label">{dayLabel(issue.date)}</span>
                <span className="truncate text-small text-dim">
                  {issue.title}
                </span>
              </h3>
              <ul className="flex flex-col gap-1 py-3">
                {issue.items.map((item) => (
                  <NewsItemRow key={item.id} item={item} />
                ))}
              </ul>
            </section>
          ))}
        </div>
      </div>
    </section>
  );
}

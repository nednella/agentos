import { useEffect, useRef } from "react";
import { useAgentos } from "../AgentosContext";
import { useLayout } from "../LayoutContext";
import { ago, useNow } from "../time";
import { Icon } from "./Icon";
import { NewsSourceRow } from "./NewsSourceRow";

const SHOWN = 3;

const dayLabel = (date: string) => {
  const [year, month, day] = date.split("-").map(Number);
  return new Date(year, month - 1, day).toLocaleDateString(undefined, {
    weekday: "long",
    day: "numeric",
    month: "short",
  });
};

export function NewsIndex() {
  const {
    news,
    newsSeen,
    newsUnseen,
    digest,
    digestSeen,
    digestUnseenCount,
    project,
    focusRequest,
  } = useAgentos();
  const { closeCentre, openNews } = useLayout();
  const now = useNow();
  const panel = useRef<HTMLElement>(null);

  useEffect(() => panel.current?.focus(), []);
  useEffect(() => {
    if (focusRequest.target === "terminal") panel.current?.focus();
  }, [focusRequest]);

  const stories = (news?.issues ?? []).flatMap((issue) =>
    issue.items.map((item) => ({
      id: item.id,
      title: item.title,
      unseen: issue.fetchedAt > newsSeen,
    })),
  );
  const newest = news?.issues[0];
  const newsSub = [
    newest ? `Daily newsletter · ${dayLabel(newest.date)}` : "No issues yet",
    news?.running && "refreshing…",
  ]
    .filter(Boolean)
    .join(" · ");
  const picks = (digest?.items ?? [])
    .slice(0, SHOWN)
    .map((item) => ({
      id: item.id,
      title: item.title,
      unseen: !item.noteId && item.at > digestSeen,
    }));
  const digestSub = [
    `From the dependencies in ${project?.name ?? "this project"}'s source code · ${digest && digest.lastRunAt > 0 ? `last run ${ago(digest.lastRunAt, now)}` : "never run"}`,
    digest?.running && "running…",
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <section
      ref={panel}
      data-panel="terminal"
      tabIndex={-1}
      aria-label="News"
      className="panel flex h-full min-h-0 min-w-0 flex-col overflow-hidden"
    >
      <header className="panel-head flex flex-none items-center gap-3 border-b border-line px-4 py-2.5">
        <h2 className="text-title font-semibold">News</h2>
        <button
          className="btn btn-ghost ml-auto h-7 w-7 justify-center px-0"
          title="Close (Esc)"
          aria-label="Close news"
          onClick={closeCentre}
        >
          <Icon name="close" />
        </button>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto flex max-w-3xl flex-col divide-y divide-line px-3 py-2">
          <NewsSourceRow
            name="Weekly digest"
            sub={digestSub}
            error={digest?.error}
            unseen={digestUnseenCount}
            headlines={picks}
            empty="Run the digest to see what changed in the tools this project's code uses."
            onOpen={() => openNews("digest")}
          />
          <NewsSourceRow
            name="TLDR Dev"
            sub={newsSub}
            error={news?.error}
            unseen={newsUnseen}
            headlines={stories.slice(0, SHOWN)}
            empty="The first fetch brings the latest issues."
            onOpen={() => openNews("tldr")}
          />
        </div>
      </div>
    </section>
  );
}

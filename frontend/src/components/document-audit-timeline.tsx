"use client";

import {
  Archive,
  ArrowCounterClockwise,
  CheckCircle,
  ClockCounterClockwise,
  Copy,
  FilePlus,
  PencilSimple,
  WarningCircle,
} from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { AuditAction, AuditEvent, documentsApi } from "@/lib/api";

type TimelineState = "loading" | "ready" | "error";

const descriptions: Record<AuditAction, { title: string; detail: string }> = {
  "document.created": { title: "Draft created", detail: "The first server-calculated version entered the ledger." },
  "document.updated": { title: "Draft updated", detail: "Editable details or line items were saved and recalculated." },
  "document.finalized": { title: "Document finalized", detail: "This version was locked as an immutable financial record." },
  "document.duplicated": { title: "Draft duplicated", detail: "A fresh draft was created from a finalized source document." },
  "document.archived": { title: "Draft archived", detail: "The draft was removed from the active ledger." },
  "document.restored": { title: "Draft restored", detail: "The draft returned to the active ledger." },
};

export function DocumentAuditTimeline({ documentId, version }: { documentId: string; version: number }) {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [state, setState] = useState<TimelineState>("loading");
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    documentsApi.auditEvents(documentId, controller.signal)
      .then((result) => { setEvents(result); setState("ready"); })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setState("error");
      });
    return () => controller.abort();
  }, [documentId, version, revision]);

  return <section className="audit-card" aria-labelledby="audit-heading" aria-live="polite">
    <header className="audit-card__header">
      <div className="audit-heading-mark" aria-hidden="true"><ClockCounterClockwise size={21} weight="duotone" /></div>
      <div><p className="eyebrow">Traceability</p><h2 id="audit-heading">Document activity</h2><p>An immutable record of material changes made to this document.</p></div>
      {state === "ready" && events.length > 0 && <span className="audit-count">{events.length} {events.length === 1 ? "event" : "events"}</span>}
    </header>

    {state === "loading" && <div className="audit-loading" aria-label="Loading document activity"><i /><i /><i /></div>}
    {state === "error" && <div className="audit-state"><WarningCircle size={24} weight="duotone" /><div><strong>Activity is unavailable</strong><p>Your document is safe. The timeline could not be loaded.</p></div><button type="button" onClick={() => { setState("loading"); setRevision((value) => value + 1); }}>Try again</button></div>}
    {state === "ready" && events.length === 0 && <div className="audit-state audit-state--empty"><ClockCounterClockwise size={24} weight="duotone" /><div><strong>No recorded activity yet</strong><p>New material changes will appear here.</p></div></div>}
    {state === "ready" && events.length > 0 && <ol className="audit-timeline">
      {events.map((event, index) => <AuditTimelineItem event={event} latest={index === 0} key={event.id} />)}
    </ol>}
  </section>;
}

function AuditTimelineItem({ event, latest }: { event: AuditEvent; latest: boolean }) {
  const copy = descriptions[event.action] ?? { title: "Document changed", detail: "A material document event was recorded." };
  return <li className="audit-event">
    <span className={`audit-event__icon audit-event__icon--${event.action.split(".")[1]}`} aria-hidden="true"><ActionIcon action={event.action} /></span>
    <div className="audit-event__body"><div><strong>{copy.title}</strong>{latest && <span className="audit-latest">Latest</span>}</div><p>{copy.detail}</p><div className="audit-event__meta"><time dateTime={event.occurredAt}>{formatTimestamp(event.occurredAt)}</time><span>Version {event.documentVersion}</span>{event.metadata.calculationPolicyVersion && <span>{event.metadata.calculationPolicyVersion}</span>}</div></div>
  </li>;
}

function ActionIcon({ action }: { action: AuditAction }) {
  if (action === "document.created") return <FilePlus size={17} weight="duotone" />;
  if (action === "document.updated") return <PencilSimple size={17} weight="duotone" />;
  if (action === "document.finalized") return <CheckCircle size={17} weight="duotone" />;
  if (action === "document.archived") return <Archive size={17} weight="duotone" />;
  if (action === "document.restored") return <ArrowCounterClockwise size={17} weight="duotone" />;
  return <Copy size={17} weight="duotone" />;
}

function formatTimestamp(value: string) {
  return new Intl.DateTimeFormat("en", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

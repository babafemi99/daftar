import { Document } from "@/lib/api";

export function StatusBadge({ document }: { document: Document }) {
  const status = document.archivedAt ? "archived" : document.status;
  return <span className={`document-status document-status--${status}`}>{status}</span>;
}

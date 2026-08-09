import { CaretLeft, CaretRight } from "@phosphor-icons/react";
import { PageMeta } from "@/lib/api";

export function DocumentPagination({ page, onChange }: { page: PageMeta; onChange: (page: number) => void }) {
  if (page.totalPages <= 1) return null;
  const first = (page.number - 1) * page.size + 1;
  const last = Math.min(page.number * page.size, page.totalItems);

  return <nav className="document-pagination" aria-label="Document pages">
    <p>Showing <strong>{first}–{last}</strong> of <strong>{page.totalItems}</strong></p>
    <div>
      <button type="button" onClick={() => onChange(page.number - 1)} disabled={page.number <= 1} aria-label="Previous page"><CaretLeft size={16} weight="bold" /> Previous</button>
      <span>Page {page.number} of {page.totalPages}</span>
      <button type="button" onClick={() => onChange(page.number + 1)} disabled={!page.hasMore} aria-label="Next page">Next <CaretRight size={16} weight="bold" /></button>
    </div>
  </nav>;
}

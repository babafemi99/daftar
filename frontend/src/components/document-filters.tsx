import { FormEvent, useState } from "react";
import { MagnifyingGlass } from "@phosphor-icons/react";
import { Currency, DocumentListFilters, DocumentStatus } from "@/lib/api";

const currencies: Currency[] = ["USD", "AED", "SAR", "NGN", "GBP", "EUR"];

export function DocumentFilters({ filters, onChange, onClear }: {
  filters: DocumentListFilters;
  onChange: (filters: DocumentListFilters) => void;
  onClear: () => void;
}) {
	const [search, setSearch] = useState(filters.search ?? "");
  const update = (key: keyof DocumentListFilters, value: string) => {
    const next = { ...filters };
    if (value === "") delete next[key];
    else if (key === "archived") next.archived = value === "true";
    else if (key === "status") next.status = value as DocumentStatus;
    else if (key === "currency") next.currency = value as Currency;
    else if (key === "from") next.from = value;
    else if (key === "to") next.to = value;
    onChange(next);
  };
	const active = Object.keys(filters).length > 0;
	const submitSearch = (event: FormEvent) => {
		event.preventDefault();
		const value = search.trim();
		if (value.length === 1) return;
		update("search", value);
	};

	return <div className="document-filters" aria-label="Document filters">
		<form className="document-search" role="search" onSubmit={submitSearch}><label htmlFor="document-search">Search documents</label><div><MagnifyingGlass size={17} aria-hidden="true" /><input id="document-search" type="search" value={search} minLength={2} maxLength={100} placeholder="Reference, title or customer" onChange={(event) => setSearch(event.target.value)} /><button type="submit">Search</button></div></form>
    <div className="filter-field"><label htmlFor="status">Status</label><select id="status" value={filters.status ?? ""} onChange={(event) => update("status", event.target.value)}><option value="">All statuses</option><option value="draft">Draft</option><option value="finalized">Finalized</option></select></div>
    <div className="filter-field"><label htmlFor="currency">Currency</label><select id="currency" value={filters.currency ?? ""} onChange={(event) => update("currency", event.target.value)}><option value="">All currencies</option>{currencies.map((currency) => <option key={currency}>{currency}</option>)}</select></div>
    <div className="filter-field"><label htmlFor="from">From</label><input id="from" type="date" value={filters.from ?? ""} onChange={(event) => update("from", event.target.value)} /></div>
    <div className="filter-field"><label htmlFor="to">To</label><input id="to" type="date" value={filters.to ?? ""} min={filters.from} onChange={(event) => update("to", event.target.value)} /></div>
    <div className="filter-field"><label htmlFor="archive">Archive</label><select id="archive" value={filters.archived === undefined ? "" : String(filters.archived)} onChange={(event) => update("archived", event.target.value)}><option value="">Active only</option><option value="true">Archived</option></select></div>
		{active && <button className="clear-filters" type="button" onClick={() => { setSearch(""); onClear(); }}>Clear</button>}
  </div>;
}

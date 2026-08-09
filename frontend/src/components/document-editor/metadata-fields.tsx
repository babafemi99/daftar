import { Currency } from "@/lib/api";
import { EditorValue } from "./editor-types";

const currencies: Currency[] = ["USD", "AED", "SAR", "NGN", "GBP", "EUR"];

export function MetadataFields({ value, errors, onChange }: { value: EditorValue; errors: Record<string, string>; onChange: (value: EditorValue) => void }) {
  const update = <K extends keyof EditorValue>(key: K, next: EditorValue[K]) => onChange({ ...value, [key]: next });
  return <section className="editor-card metadata-card"><div className="editor-section-heading"><div><span>01</span><h2>Document details</h2></div><p>Set the record’s identity and currency.</p></div>
    <div className="metadata-grid">
      <div className="editor-field editor-field--wide"><label htmlFor="document-title">Title</label><input id="document-title" data-field-path="title" aria-invalid={Boolean(errors.title)} aria-describedby={errors.title ? "error-title" : undefined} value={value.title} onChange={(event) => update("title", event.target.value)} placeholder="August software services" required />{errors.title && <p className="field-error" id="error-title">{errors.title}</p>}</div>
      <div className="editor-field editor-field--wide"><label htmlFor="customer">Customer</label><input id="customer" data-field-path="customer" aria-invalid={Boolean(errors.customer)} aria-describedby={errors.customer ? "error-customer" : undefined} value={value.customer} onChange={(event) => update("customer", event.target.value)} placeholder="Acme Limited" required />{errors.customer && <p className="field-error" id="error-customer">{errors.customer}</p>}</div>
      <div className="editor-field"><label htmlFor="issue-date">Issue date</label><input id="issue-date" data-field-path="issueDate" aria-invalid={Boolean(errors.issueDate)} aria-describedby={errors.issueDate ? "error-issue-date" : undefined} type="date" value={value.issueDate} onChange={(event) => update("issueDate", event.target.value)} required />{errors.issueDate && <p className="field-error" id="error-issue-date">{errors.issueDate}</p>}</div>
      <div className="editor-field"><label htmlFor="document-currency">Currency</label><select id="document-currency" data-field-path="currency" aria-invalid={Boolean(errors.currency)} aria-describedby={errors.currency ? "error-currency" : undefined} value={value.currency} onChange={(event) => update("currency", event.target.value as Currency)}>{currencies.map((currency) => <option key={currency}>{currency}</option>)}</select>{errors.currency && <p className="field-error" id="error-currency">{errors.currency}</p>}</div>
    </div>
  </section>;
}

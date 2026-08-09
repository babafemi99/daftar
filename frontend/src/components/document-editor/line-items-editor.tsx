"use client";

import { useEffect } from "react";
import { EditableLine } from "./editor-types";
import { Currency } from "@/lib/api";
import { ArrowDown, ArrowUp, Copy, Plus, Trash } from "@phosphor-icons/react";

export function LineItemsEditor({ lines, totals, errors, onChange, onAdd, currency }: {
  lines: EditableLine[];
  totals: string[];
  errors: Record<string, string>;
  onChange: (lines: EditableLine[]) => void;
  onAdd: () => void;
  currency: Currency;
}) {
  const update = (index: number, patch: Partial<EditableLine>) => onChange(lines.map((line, position) => position === index ? { ...line, ...patch } : line));
  const remove = (index: number) => onChange(lines.filter((_, position) => position !== index));
  const move = (index: number, direction: -1 | 1) => { const target = index + direction; if (target < 0 || target >= lines.length) return; const next = [...lines]; [next[index], next[target]] = [next[target], next[index]]; onChange(next); };
  const focusDescription = (key: string) => window.requestAnimationFrame(() => document.getElementById(`description-${key}`)?.focus());
  const add = () => { onAdd(); window.requestAnimationFrame(() => document.querySelector<HTMLInputElement>(".line-editor:last-child .line-description input")?.focus()); };
  const duplicate = (index: number) => { const source = lines[index]; const copy = { ...source, key: crypto.randomUUID(), id: undefined }; const next = [...lines]; next.splice(index + 1, 0, copy); onChange(next); focusDescription(copy.key); };
  const errorProps = (path: string, id: string) => ({ "data-field-path": path, "aria-invalid": Boolean(errors[path]), "aria-describedby": errors[path] ? id : undefined });
  const errorMessage = (path: string, id: string) => errors[path] ? <p className="field-error" id={id}>{errors[path]}</p> : null;

  useEffect(() => {
    const shortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key === "Enter") { event.preventDefault(); add(); }
    };
    window.addEventListener("keydown", shortcut);
    return () => window.removeEventListener("keydown", shortcut);
  });

  return <section className="editor-card lines-card"><div className="editor-section-heading"><div><span>02</span><h2>Line items</h2></div><p>Discounts and taxes are calculated per line.</p></div>
    <div className="line-list">{lines.map((line, index) => <article className="line-editor" aria-label={`Line item ${index + 1}`} key={line.key}>
      <div className="line-editor__top"><span className="line-number">Line {String(index + 1).padStart(2, "0")}</span><div className="line-controls"><button type="button" aria-label={`Duplicate line ${index + 1}`} title="Duplicate line" onClick={() => duplicate(index)}><Copy size={14} /></button><button type="button" aria-label="Move line up" title="Move up" disabled={index === 0} onClick={() => move(index, -1)}><ArrowUp size={14} weight="bold" /></button><button type="button" aria-label="Move line down" title="Move down" disabled={index === lines.length - 1} onClick={() => move(index, 1)}><ArrowDown size={14} weight="bold" /></button><button type="button" className="remove-line" aria-label="Remove line" title="Remove line" onClick={() => remove(index)}><Trash size={15} weight="regular" /></button></div></div>
      <div className="line-grid">
        <div className="editor-field line-description"><label htmlFor={`description-${line.key}`}>Description</label><input id={`description-${line.key}`} {...errorProps(`lineItems[${index}].description`, `error-description-${line.key}`)} value={line.description} onChange={(event) => update(index, { description: event.target.value })} placeholder="Consulting services" required />{errorMessage(`lineItems[${index}].description`, `error-description-${line.key}`)}</div>
        <div className="editor-field"><label htmlFor={`quantity-${line.key}`}>Quantity</label><input id={`quantity-${line.key}`} {...errorProps(`lineItems[${index}].quantity`, `error-quantity-${line.key}`)} type="number" inputMode="numeric" min="1" step="1" value={line.quantity} onChange={(event) => update(index, { quantity: event.target.value })} required />{errorMessage(`lineItems[${index}].quantity`, `error-quantity-${line.key}`)}</div>
        <div className="editor-field"><label htmlFor={`price-${line.key}`}>Unit price</label><div className="money-input"><span>{currency}</span><input id={`price-${line.key}`} {...errorProps(`lineItems[${index}].unitPrice`, `error-price-${line.key}`)} inputMode="decimal" value={line.unitPrice} onChange={(event) => update(index, { unitPrice: event.target.value })} required /></div>{errorMessage(`lineItems[${index}].unitPrice`, `error-price-${line.key}`)}</div>
        <div className="editor-field"><label htmlFor={`discount-${line.key}`}>Discount</label><select id={`discount-${line.key}`} {...errorProps(`lineItems[${index}].discount.type`, `error-discount-type-${line.key}`)} value={line.discountType} onChange={(event) => update(index, { discountType: event.target.value as EditableLine["discountType"], discountValue: "" })}><option value="none">None</option><option value="fixed">Fixed</option><option value="percentage">Percentage</option></select>{errorMessage(`lineItems[${index}].discount.type`, `error-discount-type-${line.key}`)}</div>
        <div className="editor-field"><label htmlFor={`discount-value-${line.key}`}>Discount value</label><input id={`discount-value-${line.key}`} {...errorProps(`lineItems[${index}].discount.value`, `error-discount-value-${line.key}`)} inputMode="decimal" value={line.discountValue} disabled={line.discountType === "none"} required={line.discountType !== "none"} placeholder={line.discountType === "percentage" ? "10" : "0.00"} onChange={(event) => update(index, { discountValue: event.target.value })} />{errorMessage(`lineItems[${index}].discount.value`, `error-discount-value-${line.key}`)}</div>
        <div className="editor-field"><label htmlFor={`tax-${line.key}`}>Tax rate %</label><input id={`tax-${line.key}`} {...errorProps(`lineItems[${index}].taxRate`, `error-tax-${line.key}`)} inputMode="decimal" value={line.taxRate} onChange={(event) => update(index, { taxRate: event.target.value })} required />{errorMessage(`lineItems[${index}].taxRate`, `error-tax-${line.key}`)}</div>
        <div className="line-total"><span>Line total</span><strong>{totals[index] ?? "—"}</strong></div>
      </div>
    </article>)}</div>
    <button className="add-line-button" type="button" onClick={add}><Plus size={16} weight="bold" /> Add line item <kbd>⌘/Ctrl ↵</kbd></button>
  </section>;
}

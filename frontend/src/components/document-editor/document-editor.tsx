"use client";

import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { Check, CloudArrowUp, WarningCircle } from "@phosphor-icons/react";
import { useRouter } from "next/navigation";
import { ApiClientError, CalculationPreview, Document as DaftarDocument, documentsApi, LineInput } from "@/lib/api";
import { useToast } from "@/components/toast-provider";
import { CalculationSummary } from "./calculation-summary";
import { EditorValue, newLine } from "./editor-types";
import { LineItemsEditor } from "./line-items-editor";
import { MetadataFields } from "./metadata-fields";

const today = new Date().toISOString().slice(0, 10);

export function DocumentEditor({ document, onSaved, formId = "document-editor-form" }: { document?: DaftarDocument; onSaved?: (document: DaftarDocument) => void; formId?: string }) {
  const router = useRouter();
  const { showToast } = useToast();
  const [value, setValue] = useState<EditorValue>(() => document ? {
    title: document.title, customer: document.customer, issueDate: document.issueDate.slice(0, 10), currency: document.currency,
    lines: document.lineItems.map((line) => ({ key: line.id, id: line.id, description: line.description, quantity: String(line.quantity), unitPrice: line.unitPrice, discountType: line.discount?.type ?? "none", discountValue: line.discount?.value ?? "", taxRate: line.taxRate })),
  } : { title: "", customer: "", issueDate: today, currency: "USD", lines: [newLine()] });
  const [preview, setPreview] = useState<CalculationPreview | null>(null);
  const [calculating, setCalculating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const rawLines = useMemo<LineInput[]>(() => value.lines.map((line) => ({ description: line.description, quantity: Number(line.quantity), unitPrice: line.unitPrice, taxRate: line.taxRate, ...(line.discountType !== "none" ? { discount: { type: line.discountType, value: line.discountValue } } : {}) })), [value.lines]);
  const persistedLines = useMemo<LineInput[]>(() => rawLines.map((line, index) => ({ ...line, ...(value.lines[index].id ? { id: value.lines[index].id } : {}) })), [rawLines, value.lines]);
  const inputsValid = !rawLines.some((line) => !line.description.trim() || !line.unitPrice || line.taxRate === "" || !Number.isInteger(line.quantity) || line.quantity < 1 || (line.discount && !line.discount.value));
  const documentValid = Boolean(value.title.trim() && value.customer.trim() && value.issueDate && inputsValid);
  const visiblePreview = inputsValid ? preview : null;
  const initialFingerprint = useMemo(() => JSON.stringify(document ? {
    title: document.title, customer: document.customer, issueDate: document.issueDate.slice(0, 10), currency: document.currency,
    lines: document.lineItems.map((line) => ({ id: line.id, description: line.description, quantity: String(line.quantity), unitPrice: line.unitPrice, discountType: line.discount?.type ?? "none", discountValue: line.discount?.value ?? "", taxRate: line.taxRate })),
  } : { title: "", customer: "", issueDate: today, currency: "USD", lines: [{ description: "", quantity: "1", unitPrice: "0.00", discountType: "none", discountValue: "", taxRate: "0" }] }), [document]);
  const currentFingerprint = JSON.stringify({ ...value, lines: value.lines.map((line) => ({ ...(line.id ? { id: line.id } : {}), description: line.description, quantity: line.quantity, unitPrice: line.unitPrice, discountType: line.discountType, discountValue: line.discountValue, taxRate: line.taxRate })) });
  const dirty = currentFingerprint !== initialFingerprint;

  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => { event.preventDefault(); };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  useEffect(() => {
    if (!inputsValid) return;
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      setCalculating(true);
      documentsApi.preview(value.currency, rawLines, controller.signal).then(setPreview).catch((cause) => { if (!(cause instanceof DOMException && cause.name === "AbortError")) setPreview(null); }).finally(() => setCalculating(false));
    }, 350);
    return () => { window.clearTimeout(timer); controller.abort(); };
  }, [inputsValid, rawLines, value.currency]);

  const persist = useCallback(async (notify: boolean) => {
    if (saving) return;
    setSaving(true);
    setSaveError(false);
    setFieldErrors({});
    try {
      const saved = document
        ? await documentsApi.replace(document.id, document.version, { title: value.title, customer: value.customer, issueDate: value.issueDate, currency: value.currency, lineItems: persistedLines })
        : await documentsApi.create({ title: value.title, customer: value.customer, issueDate: value.issueDate, currency: value.currency, lineItems: rawLines });
      if (notify || !document) showToast({ tone: "success", title: document ? "Draft saved" : "Draft created", message: `${saved.reference} is up to date.` });
      onSaved?.(saved);
      if (!document) router.push(`/documents/${saved.id}`);
    } catch (cause) {
      const error = cause instanceof ApiClientError ? cause : null;
      setSaveError(true);
      const nextErrors = Object.fromEntries(error?.fields.map((field) => [field.path, field.message]) ?? []);
      setFieldErrors(nextErrors);
      const firstPath = error?.fields[0]?.path;
      if ((notify || !document) && firstPath) window.requestAnimationFrame(() => window.document.getElementById(formId)?.querySelector<HTMLElement>(`[data-field-path="${firstPath}"]`)?.focus());
      showToast({ tone: "error", title: document ? "Couldn’t save draft" : "Couldn’t create draft", message: error?.code === "DOCUMENT_VERSION_CONFLICT" ? "This document changed elsewhere. Refresh before saving again." : error?.fields[0]?.message ?? error?.message ?? "We could not reach Daftar." });
    } finally { setSaving(false); }
  }, [document, formId, onSaved, persistedLines, rawLines, router, saving, showToast, value.currency, value.customer, value.issueDate, value.title]);

  useEffect(() => {
    if (!document || !dirty || !documentValid || saving || saveError) return;
    const timer = window.setTimeout(() => void persist(false), 1100);
    return () => window.clearTimeout(timer);
  }, [dirty, document, documentValid, persist, saveError, saving]);

  function save(event: FormEvent) { event.preventDefault(); void persist(true); }

  const saveState = saveError ? <><WarningCircle size={14} /> Save failed — try again</> : saving ? <><CloudArrowUp size={14} /> Saving changes…</> : dirty ? <><span className="save-state-dot" /> Unsaved changes</> : <><Check size={14} weight="bold" /> All changes saved</>;

  return <form id={formId} className="document-editor" noValidate onSubmit={save}><div className="editor-main"><MetadataFields value={value} errors={fieldErrors} onChange={(next) => { setSaveError(false); setFieldErrors({}); setValue(next); }} /><LineItemsEditor lines={value.lines} errors={fieldErrors} currency={value.currency} totals={visiblePreview?.lineItems.map((line) => `${value.currency} ${line.calculated.lineTotal}`) ?? []} onChange={(lines) => { setSaveError(false); setFieldErrors({}); setValue({ ...value, lines }); }} onAdd={() => setValue({ ...value, lines: [...value.lines, newLine()] })} /></div><div className="editor-side"><div className={`save-state ${saveError ? "save-state--error" : dirty ? "save-state--dirty" : ""}`} aria-live="polite">{document ? saveState : "A reference is assigned when you create this draft."}</div><CalculationSummary preview={visiblePreview} currency={value.currency} calculating={calculating} /><p className="save-hint">Drafts autosave after valid changes. Finalization remains explicit.</p></div></form>;
}

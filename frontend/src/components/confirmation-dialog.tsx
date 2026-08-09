"use client";

import { Archive, CheckCircle } from "@phosphor-icons/react";
import { useEffect, useRef } from "react";

export function ConfirmationDialog({ open, title, message, confirmLabel, tone = "primary", busy, onConfirm, onClose }: {
  open: boolean; title: string; message: string; confirmLabel: string; tone?: "primary" | "danger"; busy?: boolean;
  onConfirm: () => void; onClose: () => void;
}) {
	const dialogRef = useRef<HTMLElement>(null);
	const returnFocusRef = useRef<HTMLElement | null>(null);
	useEffect(() => {
		if (!open) return;
		returnFocusRef.current = document.activeElement as HTMLElement | null;
		const dialog = dialogRef.current;
		const focusable = () => Array.from(dialog?.querySelectorAll<HTMLElement>('button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])') ?? []);
		focusable()[0]?.focus();
		const keyboard = (event: KeyboardEvent) => {
			if (event.key === "Escape" && !busy) { event.preventDefault(); onClose(); return; }
			if (event.key !== "Tab") return;
			const items = focusable();
			if (!items.length) return;
			const first = items[0], last = items[items.length - 1];
			if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
			else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
		};
		document.addEventListener("keydown", keyboard);
		return () => { document.removeEventListener("keydown", keyboard); returnFocusRef.current?.focus(); };
	}, [busy, onClose, open]);
  if (!open) return null;
  return <div className="dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}>
    <section ref={dialogRef} className="confirmation-dialog" role="alertdialog" aria-modal="true" aria-labelledby="dialog-title" aria-describedby="dialog-message">
      <span className={`dialog-mark dialog-mark--${tone}`} aria-hidden="true">{tone === "danger" ? <Archive size={20} weight="duotone" /> : <CheckCircle size={21} weight="duotone" />}</span>
      <h2 id="dialog-title">{title}</h2><p id="dialog-message">{message}</p>
      <div className="dialog-actions"><button type="button" className="dialog-cancel" onClick={onClose} disabled={busy}>Cancel</button><button type="button" className={tone === "danger" ? "dialog-danger" : "dialog-confirm"} onClick={onConfirm} disabled={busy}>{busy ? "Working…" : confirmLabel}</button></div>
    </section>
  </div>;
}

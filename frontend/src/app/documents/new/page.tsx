"use client";

import Link from "next/link";
import { ArrowLeft, FloppyDisk } from "@phosphor-icons/react";
import { AppShell } from "@/components/app-shell";
import { DocumentEditor } from "@/components/document-editor/document-editor";
import { Protected } from "@/components/session-provider";

export default function NewDocumentPage() {
  return <Protected><AppShell><main id="main-content" className="editor-page" tabIndex={-1}><div className="editor-page-heading"><div><Link href="/documents"><ArrowLeft size={15} weight="bold" /> Documents</Link><p className="eyebrow">New record</p><h1>New document</h1></div><button className="header-save-button" type="submit" form="new-document-form"><FloppyDisk size={18} weight="bold" /> Create draft</button></div><DocumentEditor formId="new-document-form" /></main></AppShell></Protected>;
}

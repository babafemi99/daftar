import Image from "next/image";

export function BrandLoadingScreen() {
  return <main className="brand-loading" aria-live="polite" aria-label="Opening Daftar">
    <div className="brand-loading__mark"><Image src="/daftar-symbol.svg" width={180} height={180} alt="" priority /><span className="brand-loading__sweep" /></div>
    <div className="brand-loading__word"><strong>Daftar</strong><span>دفتر</span></div>
    <div className="brand-loading__lines" aria-hidden="true"><i /><i /><i /></div>
    <p>Opening your ledger…</p>
  </main>;
}

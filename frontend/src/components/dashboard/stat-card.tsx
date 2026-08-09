import { Icon } from "@phosphor-icons/react";

export function StatCard({ label, value, detail, icon: IconComponent, tone = "green" }: { label: string; value: string | number; detail: string; icon: Icon; tone?: "green" | "copper" | "ink" }) {
  return <article className={`dashboard-stat dashboard-stat--${tone}`}><span><IconComponent size={20} weight="duotone" /></span><div><p>{label}</p><strong>{value}</strong><small>{detail}</small></div></article>;
}

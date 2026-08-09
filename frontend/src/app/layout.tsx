import type { Metadata } from "next";
import { ToastProvider } from "@/components/toast-provider";
import { SessionProvider } from "@/components/session-provider";
import "./globals.css";

export const metadata: Metadata = {
  title: "Daftar — Your business, clearly recorded",
  description: "Create precise, multi-rate business documents with confidence.",
  icons: { icon: "/favicon.svg" },
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body><ToastProvider><SessionProvider>{children}</SessionProvider></ToastProvider></body>
    </html>
  );
}

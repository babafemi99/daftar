"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AuthShell } from "@/components/auth-shell";
import { useToast } from "@/components/toast-provider";
import { GuestOnly, useSession } from "@/components/session-provider";
import { ApiClientError, authApi } from "@/lib/api";

export default function RegisterPage() {
  const router = useRouter();
  const { showToast } = useToast();
  const { establishSession } = useSession();
  const [loading, setLoading] = useState(false);

  async function register(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    const form = new FormData(event.currentTarget);
    try {
      const user = await authApi.register({
        first_name: String(form.get("first_name")), last_name: String(form.get("last_name")),
        email: String(form.get("email")), password: String(form.get("password")),
      });
      establishSession(user);
      showToast({ tone: "success", title: "Account created", message: "Welcome to Daftar. Your ledger is ready." });
      router.push("/dashboard");
    } catch (cause) {
      const error = cause instanceof ApiClientError ? cause : null;
      const fieldMessage = error?.fields[0]?.message;
      showToast({ tone: "error", title: "Couldn’t create account", message: fieldMessage ?? error?.message ?? "We could not reach Daftar." });
    } finally { setLoading(false); }
  }

  return <GuestOnly><AuthShell>
    <p className="eyebrow">Start clearly</p>
    <h2>Create your ledger</h2>
    <p className="intro">A calm place for precise business documents.</p>
    <form onSubmit={register}>
      <div className="field-row">
        <div><label htmlFor="first_name">First name</label><input id="first_name" name="first_name" autoComplete="given-name" required /></div>
        <div><label htmlFor="last_name">Last name</label><input id="last_name" name="last_name" autoComplete="family-name" required /></div>
      </div>
      <label htmlFor="email">Email address</label>
      <input id="email" name="email" type="email" placeholder="you@company.com" autoComplete="email" required />
      <label htmlFor="password">Password</label>
      <input id="password" name="password" type="password" placeholder="At least 8 characters" autoComplete="new-password" minLength={8} required />
      <button type="submit" disabled={loading}>{loading ? "Creating account…" : "Create account"}</button>
    </form>
    <p className="signup">Already have an account? <Link href="/">Sign in</Link></p>
    <p className="legal">By creating an account, you agree to keep your account credentials secure.</p>
  </AuthShell></GuestOnly>;
}

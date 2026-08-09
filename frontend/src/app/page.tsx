"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AuthShell } from "@/components/auth-shell";
import { useToast } from "@/components/toast-provider";
import { GuestOnly, useSession } from "@/components/session-provider";
import { ApiClientError, authApi } from "@/lib/api";

export default function Home() {
  const router = useRouter();
  const { showToast } = useToast();
  const { establishSession } = useSession();
  const [loading, setLoading] = useState(false);

  async function signIn(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    const form = new FormData(event.currentTarget);
    try {
      const user = await authApi.login({ email: String(form.get("email")), password: String(form.get("password")) });
      establishSession(user);
      showToast({ tone: "success", title: "Welcome back", message: "Your ledger is ready." });
      router.push("/dashboard");
    } catch (cause) {
      showToast({ tone: "error", title: "Couldn’t sign in", message: cause instanceof ApiClientError ? cause.message : "We could not reach Daftar." });
    } finally { setLoading(false); }
  }

  return <GuestOnly><AuthShell>
    <p className="eyebrow">Welcome back</p>
    <h2>Sign in to your ledger</h2>
    <p className="intro">Use your Daftar account to continue.</p>
    <form onSubmit={signIn}>
      <label htmlFor="email">Email address</label>
      <input id="email" name="email" type="email" placeholder="you@company.com" autoComplete="email" required />
      <label htmlFor="password">Password</label>
      <input id="password" name="password" type="password" placeholder="Enter your password" autoComplete="current-password" minLength={8} required />
      <button type="submit" disabled={loading}>{loading ? "Signing in…" : "Sign in"}</button>
    </form>
    <p className="signup">New to Daftar? <Link href="/register">Create an account</Link></p>
    <p className="legal">By continuing, you agree to keep your account credentials secure.</p>
  </AuthShell></GuestOnly>;
}

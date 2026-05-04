"use client";

import { FormEvent, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { registerOrganization } from "@/lib/api";
import { saveAuth } from "@/lib/auth";

export default function RegisterPage() {
  const router = useRouter();
  const [organizationName, setOrganizationName] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    const input = {
      organization_name: organizationName.trim(),
      first_name: firstName.trim(),
      last_name: lastName.trim(),
      email: email.trim(),
      password,
    };

    if (!input.organization_name || !input.first_name || !input.last_name) {
      setError("Заповніть назву організації, ім'я та прізвище.");
      return;
    }

    if (password.length < 8) {
      setError("Пароль має містити щонайменше 8 символів.");
      return;
    }

    setIsSubmitting(true);
    try {
      const auth = await registerOrganization(input);
      saveAuth(auth);
      router.replace("/tickets");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не вдалося зареєструвати організацію.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="min-h-screen bg-surface px-4 py-10">
      <section className="mx-auto w-full max-w-[520px] rounded-lg border border-border bg-white p-6 shadow-sm">
        <div className="mb-6">
          <Link className="text-sm font-medium text-primary" href="/">
            TicketFlow
          </Link>
          <h1 className="mt-2 text-2xl font-semibold text-text">
            Реєстрація організації
          </h1>
          <p className="mt-2 text-sm leading-6 text-muted">
            Після створення організації буде створено перший обліковий запис
            адміністратора цієї організації.
          </p>
        </div>

        <form className="space-y-4" onSubmit={handleSubmit}>
          <label className="block">
            <span className="mb-1 block text-sm font-medium text-text">
              Назва організації
            </span>
            <input
              className="w-full rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={organizationName}
              onChange={(event) => setOrganizationName(event.target.value)}
              maxLength={255}
              required
            />
          </label>

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-1 block text-sm font-medium text-text">Ім'я</span>
              <input
                className="w-full rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
                value={firstName}
                onChange={(event) => setFirstName(event.target.value)}
                required
              />
            </label>

            <label className="block">
              <span className="mb-1 block text-sm font-medium text-text">Прізвище</span>
              <input
                className="w-full rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
                value={lastName}
                onChange={(event) => setLastName(event.target.value)}
                required
              />
            </label>
          </div>

          <label className="block">
            <span className="mb-1 block text-sm font-medium text-text">Email</span>
            <input
              className="w-full rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              autoComplete="email"
              required
            />
          </label>

          <label className="block">
            <span className="mb-1 block text-sm font-medium text-text">Пароль</span>
            <input
              className="w-full rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="new-password"
              minLength={8}
              required
            />
          </label>

          {error ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">
              {error}
            </p>
          ) : null}

          <button
            className="w-full rounded-md bg-primary px-4 py-2 font-medium text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-65"
            type="submit"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Створення..." : "Створити організацію"}
          </button>
        </form>

        <p className="mt-5 text-center text-sm text-muted">
          Уже маєте обліковий запис?{" "}
          <Link className="font-medium text-primary" href="/login">
            Увійти
          </Link>
        </p>
      </section>
    </main>
  );
}

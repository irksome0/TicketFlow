"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";

import { acceptInvite, getInvite } from "@/lib/api";
import type { PublicInvite } from "@/lib/types";

function formatDate(value: string): string {
  return new Intl.DateTimeFormat("uk-UA", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

export default function AcceptInvitePage() {
  const router = useRouter();
  const params = useParams<{ token: string }>();
  const token = params.token;

  const [invite, setInvite] = useState<PublicInvite | null>(null);
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isAccepted, setIsAccepted] = useState(false);

  useEffect(() => {
    async function loadInvite() {
      setError(null);
      try {
        const response = await getInvite(token);
        setInvite(response);
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Не вдалося отримати запрошення.",
        );
      } finally {
        setIsLoading(false);
      }
    }

    void loadInvite();
  }, [token]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    const input = {
      first_name: firstName.trim(),
      last_name: lastName.trim(),
      password,
    };

    if (!input.first_name || !input.last_name) {
      setError("Заповніть ім'я та прізвище.");
      return;
    }
    if (password.length < 8) {
      setError("Пароль має містити щонайменше 8 символів.");
      return;
    }

    setIsSubmitting(true);
    try {
      await acceptInvite(token, input);
      setIsAccepted(true);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Не вдалося прийняти запрошення.",
      );
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
            Прийняття запрошення
          </h1>
          {invite ? (
            <p className="mt-2 text-sm leading-6 text-muted">
              Запрошення для {invite.email}. Роль у системі: {invite.role}.
              Посилання діє до {formatDate(invite.expires_at)}.
            </p>
          ) : null}
        </div>

        {isLoading ? (
          <p className="text-sm text-muted">Перевірка запрошення...</p>
        ) : null}

        {!isLoading && error && !invite ? (
          <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">
            {error}
          </div>
        ) : null}

        {isAccepted ? (
          <div className="space-y-4">
            <p className="rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-success">
              Обліковий запис створено. Тепер можна увійти з email із
              запрошення та заданим паролем.
            </p>
            <button
              className="w-full rounded-md bg-primary px-4 py-2 font-medium text-white transition hover:bg-primary/90"
              type="button"
              onClick={() => router.replace("/login")}
            >
              Перейти до входу
            </button>
          </div>
        ) : null}

        {invite && !isAccepted ? (
          <form className="space-y-4" onSubmit={handleSubmit}>
            <div className="grid gap-4 sm:grid-cols-2">
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-text">
                  Ім'я
                </span>
                <input
                  className="w-full rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
                  value={firstName}
                  onChange={(event) => setFirstName(event.target.value)}
                  autoComplete="given-name"
                  required
                />
              </label>

              <label className="block">
                <span className="mb-1 block text-sm font-medium text-text">
                  Прізвище
                </span>
                <input
                  className="w-full rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
                  value={lastName}
                  onChange={(event) => setLastName(event.target.value)}
                  autoComplete="family-name"
                  required
                />
              </label>
            </div>

            <label className="block">
              <span className="mb-1 block text-sm font-medium text-text">
                Пароль
              </span>
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
              {isSubmitting ? "Створення..." : "Створити обліковий запис"}
            </button>
          </form>
        ) : null}

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

"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { createTicket } from "@/lib/api";
import { getStoredUser, getToken } from "@/lib/auth";
import type { TicketPriority, User } from "@/lib/types";

const priorities: TicketPriority[] = ["High", "Medium", "Low"];

export default function NewTicketPage() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState<TicketPriority>("Medium");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    const token = getToken();
    const storedUser = getStoredUser();

    if (!token || !storedUser) {
      router.replace("/login");
      return;
    }

    setUser(storedUser);
  }, [router]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    const trimmedTitle = title.trim();
    const trimmedDescription = description.trim();

    if (!trimmedTitle || !trimmedDescription) {
      setError("Заповніть назву та опис заявки.");
      return;
    }

    setIsSubmitting(true);
    try {
      const ticket = await createTicket({
        title: trimmedTitle,
        description: trimmedDescription,
        priority,
      });
      router.replace(`/tickets/${ticket.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не вдалося створити заявку.");
    } finally {
      setIsSubmitting(false);
    }
  }

  if (!user) {
    return null;
  }

  return (
    <main className="min-h-screen bg-surface">
      <header className="border-b border-border bg-white">
        <div className="mx-auto flex max-w-4xl items-center justify-between gap-4 px-4 py-4">
          <div>
            <p className="text-sm font-medium text-primary">TicketFlow</p>
            <h1 className="text-xl font-semibold text-text">Нова заявка</h1>
          </div>
          <button
            className="rounded-md border border-border bg-white px-3 py-2 text-sm font-medium text-text transition hover:bg-surface"
            type="button"
            onClick={() => router.push("/tickets")}
          >
            До списку
          </button>
        </div>
      </header>

      <section className="mx-auto max-w-4xl px-4 py-6">
        <form
          className="space-y-5 rounded-lg border border-border bg-white p-5 shadow-sm"
          onSubmit={handleSubmit}
        >
          <label className="block">
            <span className="mb-1 block text-sm font-medium text-text">Назва</span>
            <input
              className="w-full rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              maxLength={255}
              required
            />
          </label>

          <label className="block">
            <span className="mb-1 block text-sm font-medium text-text">Опис</span>
            <textarea
              className="min-h-36 w-full resize-y rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              required
            />
          </label>

          <label className="block">
            <span className="mb-1 block text-sm font-medium text-text">Пріоритет</span>
            <select
              className="w-full rounded-md border border-border bg-white px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={priority}
              onChange={(event) => setPriority(event.target.value as TicketPriority)}
            >
              {priorities.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </label>

          {error ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">
              {error}
            </p>
          ) : null}

          <div className="flex justify-end gap-3">
            <button
              className="rounded-md border border-border bg-white px-4 py-2 text-sm font-medium text-text transition hover:bg-surface"
              type="button"
              onClick={() => router.push("/tickets")}
            >
              Скасувати
            </button>
            <button
              className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-65"
              type="submit"
              disabled={isSubmitting}
            >
              {isSubmitting ? "Створення..." : "Створити"}
            </button>
          </div>
        </form>
      </section>
    </main>
  );
}

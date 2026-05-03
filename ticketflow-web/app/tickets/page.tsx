"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { getTickets } from "@/lib/api";
import { clearAuth, getStoredUser, getToken } from "@/lib/auth";
import type { Ticket, TicketPriority, TicketStatus, User } from "@/lib/types";

function statusClass(status: TicketStatus): string {
  switch (status) {
    case "New":
      return "border-blue-200 bg-blue-50 text-blue-700";
    case "In Progress":
      return "border-cyan-200 bg-cyan-50 text-primary";
    case "Resolved":
      return "border-green-200 bg-green-50 text-success";
    case "Closed":
      return "border-slate-200 bg-slate-100 text-slate-700";
    case "Reopened":
      return "border-amber-200 bg-amber-50 text-warning";
  }
}

function priorityClass(priority: TicketPriority): string {
  switch (priority) {
    case "High":
      return "border-red-200 bg-red-50 text-danger";
    case "Medium":
      return "border-amber-200 bg-amber-50 text-warning";
    case "Low":
      return "border-green-200 bg-green-50 text-success";
  }
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat("uk-UA", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

export default function TicketsPage() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const userName = useMemo(() => {
    if (!user) {
      return "";
    }
    return `${user.first_name} ${user.last_name}`.trim() || user.email;
  }, [user]);

  useEffect(() => {
    const token = getToken();
    const storedUser = getStoredUser();

    if (!token || !storedUser) {
      router.replace("/login");
      return;
    }

    setUser(storedUser);

    async function loadTickets() {
      try {
        const response = await getTickets();
        setTickets(response.data);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Помилка отримання заявок";
        setError(message);
        if (message.toLowerCase().includes("unauthorized")) {
          router.replace("/login");
        }
      } finally {
        setIsLoading(false);
      }
    }

    void loadTickets();
  }, [router]);

  function handleLogout() {
    clearAuth();
    router.replace("/login");
  }

  if (!user) {
    return null;
  }

  return (
    <main className="min-h-screen bg-surface">
      <header className="border-b border-border bg-white">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4">
          <div>
            <p className="text-sm font-medium text-primary">TicketFlow</p>
            <h1 className="text-xl font-semibold text-text">Заявки</h1>
          </div>
          <div className="flex items-center gap-3">
            <div className="hidden text-right sm:block">
              <p className="text-sm font-medium text-text">{userName}</p>
              <p className="text-xs text-muted">{user.role}</p>
            </div>
            <button
              className="rounded-md border border-border bg-white px-3 py-2 text-sm font-medium text-text transition hover:bg-surface"
              type="button"
              onClick={handleLogout}
            >
              Вийти
            </button>
          </div>
        </div>
      </header>

      <section className="mx-auto max-w-6xl px-4 py-6">
        <div className="mb-4 flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
          <div>
            <h2 className="text-lg font-semibold text-text">Список заявок</h2>
            <p className="text-sm text-muted">Усього: {tickets.length}</p>
          </div>
          {user.role === "client" ? (
            <button
              className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-white transition hover:bg-primary/90"
              type="button"
              onClick={() => router.push("/tickets/new")}
            >
              Нова заявка
            </button>
          ) : null}
        </div>

        {error ? (
          <p className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">
            {error}
          </p>
        ) : null}

        <div className="overflow-hidden rounded-lg border border-border bg-white">
          <div className="overflow-x-auto">
            <table className="min-w-full border-collapse text-left text-sm">
              <thead className="bg-surface text-xs uppercase text-muted">
                <tr>
                  <th className="px-4 py-3 font-semibold">Назва</th>
                  <th className="px-4 py-3 font-semibold">Статус</th>
                  <th className="px-4 py-3 font-semibold">Пріоритет</th>
                  <th className="px-4 py-3 font-semibold">Створено</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {isLoading ? (
                  <tr>
                    <td className="px-4 py-6 text-muted" colSpan={4}>
                      Завантаження...
                    </td>
                  </tr>
                ) : tickets.length === 0 ? (
                  <tr>
                    <td className="px-4 py-6 text-muted" colSpan={4}>
                      Заявок не знайдено
                    </td>
                  </tr>
                ) : (
                  tickets.map((ticket) => (
                    <tr
                      className="cursor-pointer transition hover:bg-surface"
                      key={ticket.id}
                      onClick={() => router.push(`/tickets/${ticket.id}`)}
                    >
                      <td className="max-w-[360px] px-4 py-3">
                        <p className="truncate font-medium text-text">{ticket.title}</p>
                        <p className="truncate text-xs text-muted">{ticket.description}</p>
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-medium ${statusClass(ticket.status)}`}
                        >
                          {ticket.status}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-medium ${priorityClass(ticket.priority)}`}
                        >
                          {ticket.priority}
                        </span>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-muted">
                        {formatDate(ticket.created_at)}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </main>
  );
}

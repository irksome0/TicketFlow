"use client";

import { ChangeEvent, useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";

import {
  downloadAttachment,
  getAttachments,
  getTicket,
  updateTicketStatus,
  uploadAttachment,
} from "@/lib/api";
import { getStoredUser, getToken } from "@/lib/auth";
import type { Attachment, Ticket, TicketStatus, User } from "@/lib/types";

const maxUploadSize = 5 * 1024 * 1024;
const allowedExtensions = ["jpg", "jpeg", "png", "pdf", "txt", "log"];

const transitions: Record<TicketStatus, TicketStatus[]> = {
  New: ["In Progress", "Closed"],
  "In Progress": ["Resolved", "Closed"],
  Resolved: ["Closed"],
  Closed: ["Reopened"],
  Reopened: ["In Progress", "Closed"],
};

const roleTargets: Record<User["role"], TicketStatus[]> = {
  client: ["Closed", "Reopened"],
  operator: ["In Progress", "Resolved", "Closed", "Reopened"],
  engineer: ["In Progress", "Resolved"],
  admin: ["In Progress", "Resolved", "Closed", "Reopened"],
};

function formatDate(value: string | null): string {
  if (!value) {
    return "Не встановлено";
  }

  return new Intl.DateTimeFormat("uk-UA", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function formatFileSize(size: number): string {
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} КБ`;
  }
  return `${(size / 1024 / 1024).toFixed(2)} МБ`;
}

function getSlaLimitHours(ticket: Ticket): number {
  switch (ticket.priority) {
    case "High":
      return 8;
    case "Medium":
      return 24;
    case "Low":
      return 72;
  }
}

function getSlaText(ticket: Ticket): { label: string; className: string } {
  const createdAt = new Date(ticket.created_at).getTime();
  const endAt = ticket.resolved_at ? new Date(ticket.resolved_at).getTime() : Date.now();
  const elapsedHours = Math.max(0, (endAt - createdAt) / 36e5);
  const limit = getSlaLimitHours(ticket);

  if (elapsedHours <= limit) {
    return {
      label: `SLA в нормі: ${elapsedHours.toFixed(1)} год / ${limit} год`,
      className: "border-green-200 bg-green-50 text-success",
    };
  }

  return {
    label: `SLA порушено: ${elapsedHours.toFixed(1)} год / ${limit} год`,
    className: "border-red-200 bg-red-50 text-danger",
  };
}

function canUseAttachments(user: User): boolean {
  return user.role !== "admin";
}

export default function TicketDetailsPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const ticketId = params.id;

  const [user, setUser] = useState<User | null>(null);
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isUploading, setIsUploading] = useState(false);
  const [statusLoading, setStatusLoading] = useState<TicketStatus | null>(null);

  const availableStatuses = useMemo(() => {
    if (!ticket || !user) {
      return [];
    }

    const next = transitions[ticket.status] ?? [];
    const allowed = roleTargets[user.role] ?? [];
    return next.filter((status) => allowed.includes(status));
  }, [ticket, user]);

  const sla = ticket ? getSlaText(ticket) : null;

  useEffect(() => {
    const token = getToken();
    const storedUser = getStoredUser();

    if (!token || !storedUser) {
      router.replace("/login");
      return;
    }

    const authenticatedUser = storedUser;
    setUser(authenticatedUser);

    async function loadTicket() {
      try {
        const loadedTicket = await getTicket(ticketId);
        setTicket(loadedTicket);

        if (authenticatedUser.role !== "admin") {
          const loadedAttachments = await getAttachments(ticketId);
          setAttachments(loadedAttachments);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Не вдалося отримати заявку.");
      } finally {
        setIsLoading(false);
      }
    }

    void loadTicket();
  }, [router, ticketId]);

  async function handleStatusChange(status: TicketStatus) {
    setError(null);
    setStatusLoading(status);
    try {
      const updated = await updateTicketStatus(ticketId, status);
      setTicket(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не вдалося змінити статус.");
    } finally {
      setStatusLoading(null);
    }
  }

  async function handleUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";

    if (!file) {
      return;
    }

    setFileError(null);
    const extension = file.name.split(".").pop()?.toLowerCase() ?? "";

    if (!allowedExtensions.includes(extension)) {
      setFileError("Недопустимий формат файлу. Дозволено: jpg, jpeg, png, pdf, txt, log.");
      return;
    }
    if (file.size > maxUploadSize) {
      setFileError("Розмір файлу не повинен перевищувати 5 МБ.");
      return;
    }

    setIsUploading(true);
    try {
      const uploaded = await uploadAttachment(ticketId, file);
      setAttachments((current) => [uploaded, ...current]);
    } catch (err) {
      setFileError(err instanceof Error ? err.message : "Не вдалося завантажити файл.");
    } finally {
      setIsUploading(false);
    }
  }

  async function handleDownload(attachment: Attachment) {
    setFileError(null);
    try {
      await downloadAttachment(ticketId, attachment);
    } catch (err) {
      setFileError(err instanceof Error ? err.message : "Не вдалося скачати файл.");
    }
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
            <h1 className="text-xl font-semibold text-text">Деталі заявки</h1>
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

      <section className="mx-auto max-w-6xl px-4 py-6">
        {isLoading ? (
          <p className="rounded-lg border border-border bg-white p-5 text-muted">
            Завантаження...
          </p>
        ) : error ? (
          <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">
            {error}
          </p>
        ) : ticket ? (
          <div className="grid gap-5 lg:grid-cols-[1fr_340px]">
            <section className="rounded-lg border border-border bg-white p-5 shadow-sm">
              <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <h2 className="text-2xl font-semibold text-text">{ticket.title}</h2>
                  <p className="mt-2 whitespace-pre-wrap text-sm leading-6 text-muted">
                    {ticket.description}
                  </p>
                </div>
                <span className="inline-flex w-fit rounded-full border border-cyan-200 bg-cyan-50 px-3 py-1 text-sm font-medium text-primary">
                  {ticket.status}
                </span>
              </div>

              <dl className="grid gap-4 border-t border-border pt-5 sm:grid-cols-2">
                <div>
                  <dt className="text-xs uppercase text-muted">Пріоритет</dt>
                  <dd className="mt-1 font-medium text-text">{ticket.priority}</dd>
                </div>
                <div>
                  <dt className="text-xs uppercase text-muted">Створено</dt>
                  <dd className="mt-1 font-medium text-text">{formatDate(ticket.created_at)}</dd>
                </div>
                <div>
                  <dt className="text-xs uppercase text-muted">Вирішено</dt>
                  <dd className="mt-1 font-medium text-text">{formatDate(ticket.resolved_at)}</dd>
                </div>
                <div>
                  <dt className="text-xs uppercase text-muted">SLA</dt>
                  <dd className="mt-1">
                    {sla ? (
                      <span className={`inline-flex rounded-full border px-3 py-1 text-sm font-medium ${sla.className}`}>
                        {sla.label}
                      </span>
                    ) : null}
                  </dd>
                </div>
              </dl>
            </section>

            <aside className="space-y-5">
              <section className="rounded-lg border border-border bg-white p-5 shadow-sm">
                <h2 className="mb-3 text-base font-semibold text-text">Зміна статусу</h2>
                {availableStatuses.length > 0 ? (
                  <div className="flex flex-wrap gap-2">
                    {availableStatuses.map((status) => (
                      <button
                        className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-65"
                        key={status}
                        type="button"
                        disabled={statusLoading !== null}
                        onClick={() => handleStatusChange(status)}
                      >
                        {statusLoading === status ? "Оновлення..." : status}
                      </button>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted">Для поточної ролі немає доступних переходів.</p>
                )}
              </section>

              {canUseAttachments(user) ? (
                <section className="rounded-lg border border-border bg-white p-5 shadow-sm">
                  <div className="mb-3 flex items-center justify-between gap-3">
                    <h2 className="text-base font-semibold text-text">Вкладення</h2>
                    <label className="cursor-pointer rounded-md border border-border bg-white px-3 py-2 text-sm font-medium text-text transition hover:bg-surface">
                      {isUploading ? "Завантаження..." : "Додати"}
                      <input
                        className="sr-only"
                        type="file"
                        onChange={handleUpload}
                        disabled={isUploading}
                      />
                    </label>
                  </div>

                  {fileError ? (
                    <p className="mb-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">
                      {fileError}
                    </p>
                  ) : null}

                  {attachments.length === 0 ? (
                    <p className="text-sm text-muted">Вкладення відсутні.</p>
                  ) : (
                    <ul className="space-y-3">
                      {attachments.map((attachment) => (
                        <li
                          className="rounded-md border border-border px-3 py-2"
                          key={attachment.id}
                        >
                          <p className="truncate text-sm font-medium text-text">
                            {attachment.file_name}
                          </p>
                          <div className="mt-2 flex items-center justify-between gap-3">
                            <span className="text-xs text-muted">
                              {formatFileSize(attachment.file_size)}
                            </span>
                            <button
                              className="rounded-md border border-border bg-white px-2.5 py-1.5 text-xs font-medium text-text transition hover:bg-surface"
                              type="button"
                              onClick={() => handleDownload(attachment)}
                            >
                              Скачати
                            </button>
                          </div>
                        </li>
                      ))}
                    </ul>
                  )}
                </section>
              ) : null}
            </aside>
          </div>
        ) : null}
      </section>
    </main>
  );
}

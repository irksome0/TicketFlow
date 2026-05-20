"use client";

import { ChangeEvent, FormEvent, useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";

import {
  createTicketComment,
  downloadAttachment,
  getAttachments,
  getTicket,
  getTicketComments,
  updateTicketStatus,
  uploadAttachment,
} from "@/lib/api";
import { getStoredUser, getToken } from "@/lib/auth";
import type { Attachment, SlaStatus, Ticket, TicketComment, TicketStatus, User } from "@/lib/types";

const maxUploadSize = 25 * 1024 * 1024;
const allowedExtensions = ["jpg", "jpeg", "png", "pdf", "txt", "log"];

const transitions: Record<TicketStatus, TicketStatus[]> = {
  New: ["In Progress", "Pending", "Closed"],
  "In Progress": ["Pending", "Waiting for Customer", "On Hold", "Resolved", "Closed"],
  Pending: ["In Progress", "Waiting for Customer", "On Hold", "Closed"],
  "Waiting for Customer": ["In Progress", "On Hold", "Closed"],
  "On Hold": ["In Progress", "Pending", "Closed"],
  Resolved: ["Closed"],
  Closed: ["Reopened"],
  Reopened: ["In Progress", "Pending", "Closed"],
};

const roleTargets: Record<User["role"], TicketStatus[]> = {
  client: ["Closed", "Reopened"],
  operator: [
    "In Progress",
    "Pending",
    "Waiting for Customer",
    "On Hold",
    "Resolved",
    "Closed",
    "Reopened",
  ],
  engineer: ["In Progress", "Pending", "Waiting for Customer", "On Hold", "Resolved"],
  admin: [
    "In Progress",
    "Pending",
    "Waiting for Customer",
    "On Hold",
    "Resolved",
    "Closed",
    "Reopened",
  ],
};

const roleLabels: Record<User["role"], string> = {
  client: "Client",
  operator: "Operator",
  engineer: "Engineer",
  admin: "Admin",
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

function formatDuration(seconds: number): string {
  const hours = seconds / 3600;
  if (hours < 1) {
    return `${Math.round(seconds / 60)} хв`;
  }
  return `${hours.toFixed(1)} год`;
}

function statusClass(status: TicketStatus): string {
  switch (status) {
    case "New":
      return "border-blue-200 bg-blue-50 text-blue-700";
    case "In Progress":
      return "border-cyan-200 bg-cyan-50 text-primary";
    case "Pending":
      return "border-amber-200 bg-amber-50 text-warning";
    case "Waiting for Customer":
      return "border-violet-200 bg-violet-50 text-violet-700";
    case "On Hold":
      return "border-slate-200 bg-slate-100 text-slate-700";
    case "Resolved":
      return "border-green-200 bg-green-50 text-success";
    case "Closed":
      return "border-slate-200 bg-slate-100 text-slate-700";
    case "Reopened":
      return "border-orange-200 bg-orange-50 text-orange-700";
  }
}

function slaClass(status: SlaStatus): string {
  switch (status) {
    case "Within SLA":
    case "Met":
      return "border-green-200 bg-green-50 text-success";
    case "Paused":
      return "border-amber-200 bg-amber-50 text-warning";
    case "Breached":
      return "border-red-200 bg-red-50 text-danger";
  }
}

function slaLabel(status: SlaStatus): string {
  switch (status) {
    case "Within SLA":
      return "SLA в нормі";
    case "Paused":
      return "SLA на паузі";
    case "Met":
      return "SLA виконано";
    case "Breached":
      return "SLA порушено";
  }
}

function getSlaText(ticket: Ticket): { label: string; className: string } {
  return {
    label: `${slaLabel(ticket.sla_status)}: ${formatDuration(ticket.active_duration_seconds)} / ${formatDuration(ticket.sla_limit_seconds)}`,
    className: slaClass(ticket.sla_status),
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
  const [comments, setComments] = useState<TicketComment[]>([]);
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [commentError, setCommentError] = useState<string | null>(null);
  const [commentMessage, setCommentMessage] = useState("");
  const [fileError, setFileError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingComments, setIsLoadingComments] = useState(false);
  const [isSubmittingComment, setIsSubmittingComment] = useState(false);
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

        setIsLoadingComments(true);
        const loadedComments = await getTicketComments(ticketId);
        setComments(loadedComments);

        if (authenticatedUser.role !== "admin") {
          const loadedAttachments = await getAttachments(ticketId);
          setAttachments(loadedAttachments);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Не вдалося отримати заявку.");
      } finally {
        setIsLoadingComments(false);
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

  async function handleCommentSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCommentError(null);

    const message = commentMessage.trim();
    if (!message) {
      setCommentError("Введіть текст коментаря.");
      return;
    }

    setIsSubmittingComment(true);
    try {
      const created = await createTicketComment(ticketId, message);
      setComments((current) => [...current, created]);
      setCommentMessage("");
    } catch (err) {
      setCommentError(err instanceof Error ? err.message : "Не вдалося додати коментар.");
    } finally {
      setIsSubmittingComment(false);
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
      setFileError("Розмір файлу не повинен перевищувати 25 МБ.");
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
                <span
                  className={`inline-flex w-fit rounded-full border px-3 py-1 text-sm font-medium ${statusClass(ticket.status)}`}
                >
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

              {ticket.status_history && ticket.status_history.length > 0 ? (
                <section className="mt-6 border-t border-border pt-5">
                  <h3 className="mb-3 text-base font-semibold text-text">Історія статусів</h3>
                  <ul className="space-y-2 text-sm">
                    {ticket.status_history.map((item) => (
                      <li className="rounded-md border border-border px-3 py-2" key={item.id}>
                        <span className="font-medium text-text">
                          {item.from_status ?? "Створено"} {"->"} {item.to_status}
                        </span>
                        <span className="ml-2 text-muted">{formatDate(item.changed_at)}</span>
                      </li>
                    ))}
                  </ul>
                </section>
              ) : null}
            </section>

            <aside className="space-y-5">
              <section className="rounded-lg border border-border bg-white p-5 shadow-sm">
                <h2 className="mb-3 text-base font-semibold text-text">Коментарі</h2>

                <form className="mb-4 space-y-3" onSubmit={handleCommentSubmit}>
                  <textarea
                    className="min-h-24 w-full resize-y rounded-md border border-border px-3 py-2 text-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
                    value={commentMessage}
                    onChange={(event) => setCommentMessage(event.target.value)}
                    placeholder="Додайте коментар до заявки"
                    disabled={isSubmittingComment}
                  />
                  {commentError ? (
                    <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">
                      {commentError}
                    </p>
                  ) : null}
                  <button
                    className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-65"
                    type="submit"
                    disabled={isSubmittingComment}
                  >
                    {isSubmittingComment ? "Надсилання..." : "Додати коментар"}
                  </button>
                </form>

                {isLoadingComments ? (
                  <p className="text-sm text-muted">Завантаження коментарів...</p>
                ) : comments.length === 0 ? (
                  <p className="text-sm text-muted">Коментарі відсутні.</p>
                ) : (
                  <ul className="space-y-3">
                    {comments.map((comment) => (
                      <li className="rounded-md border border-border px-3 py-2" key={comment.id}>
                        <div className="mb-1 flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
                          <span className="font-medium text-text">
                            {comment.author_name}
                            {comment.author_id === user.id ? " (Ви)" : ""}
                            <span className="ml-2 rounded-full border border-border bg-surface px-2 py-0.5 text-[11px] font-medium text-muted">
                              {roleLabels[comment.author_role]}
                            </span>
                            {comment.author_id === ticket.creator_id ? (
                              <span className="ml-2 rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-[11px] font-medium text-blue-700">
                                Автор заявки
                              </span>
                            ) : null}
                          </span>
                          <span>{formatDate(comment.created_at)}</span>
                        </div>
                        <p className="whitespace-pre-wrap text-sm leading-6 text-text">
                          {comment.message}
                        </p>
                      </li>
                    ))}
                  </ul>
                )}
              </section>

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
                  <p className="text-sm text-muted">
                    Для поточної ролі немає доступних переходів.
                  </p>
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

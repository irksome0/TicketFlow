"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { createInvite, getUsers, updateUserRole } from "@/lib/api";
import { getStoredUser, getToken } from "@/lib/auth";
import type { Invite, Role, User } from "@/lib/types";

const roles: Role[] = ["client", "operator", "engineer", "admin"];

const roleLabels: Record<Role, string> = {
  client: "Client",
  operator: "Operator",
  engineer: "Engineer",
  admin: "Admin",
};

function formatDate(value?: string): string {
  if (!value) {
    return "-";
  }

  return new Intl.DateTimeFormat("uk-UA", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function normalizeInviteUrl(invite: Invite): string {
  if (!invite.invite_url) {
    return "";
  }
  if (invite.invite_url.startsWith("http")) {
    return invite.invite_url;
  }
  return `${window.location.origin}${invite.invite_url}`;
}

export default function UsersPage() {
  const router = useRouter();
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<Role>("client");
  const [createdInvite, setCreatedInvite] = useState<Invite | null>(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreatingInvite, setIsCreatingInvite] = useState(false);
  const [updatingUserId, setUpdatingUserId] = useState<string | null>(null);

  useEffect(() => {
    const token = getToken();
    const storedUser = getStoredUser();

    if (!token || !storedUser) {
      router.replace("/login");
      return;
    }

    if (storedUser.role !== "admin") {
      router.replace("/tickets");
      return;
    }

    setCurrentUser(storedUser);

    async function loadUsers() {
      try {
        const response = await getUsers();
        setUsers(response.data);
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Не вдалося отримати користувачів.",
        );
      } finally {
        setIsLoading(false);
      }
    }

    void loadUsers();
  }, [router]);

  async function handleInviteSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setCopied(false);
    setCreatedInvite(null);

    const email = inviteEmail.trim().toLowerCase();
    if (!email) {
      setError("Вкажіть email користувача.");
      return;
    }

    setIsCreatingInvite(true);
    try {
      const invite = await createInvite({ email, role: inviteRole });
      setCreatedInvite(invite);
      setInviteEmail("");
      setInviteRole("client");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Не вдалося створити запрошення.",
      );
    } finally {
      setIsCreatingInvite(false);
    }
  }

  async function handleCopyInvite() {
    if (!createdInvite) {
      return;
    }

    const inviteUrl = normalizeInviteUrl(createdInvite);
    await navigator.clipboard.writeText(inviteUrl);
    setCopied(true);
  }

  async function handleRoleChange(userId: string, role: Role) {
    setError(null);
    setUpdatingUserId(userId);

    try {
      const updated = await updateUserRole(userId, role);
      setUsers((current) =>
        current.map((user) => (user.id === updated.id ? updated : user)),
      );
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Не вдалося оновити роль.",
      );
    } finally {
      setUpdatingUserId(null);
    }
  }

  if (!currentUser) {
    return null;
  }

  const inviteUrl = createdInvite ? normalizeInviteUrl(createdInvite) : "";

  return (
    <main className="min-h-screen bg-surface">
      <header className="border-b border-border bg-white">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4">
          <div>
            <p className="text-sm font-medium text-primary">TicketFlow</p>
            <h1 className="text-xl font-semibold text-text">Користувачі</h1>
          </div>
          <button
            className="rounded-md border border-border bg-white px-3 py-2 text-sm font-medium text-text transition hover:bg-surface"
            type="button"
            onClick={() => router.push("/tickets")}
          >
            До заявок
          </button>
        </div>
      </header>

      <section className="mx-auto max-w-6xl px-4 py-6">
        <div className="mb-6 rounded-lg border border-border bg-white p-4 shadow-sm">
          <h2 className="text-base font-semibold text-text">
            Запросити учасника
          </h2>
          <form
            className="mt-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_180px_auto]"
            onSubmit={handleInviteSubmit}
          >
            <label className="block">
              <span className="mb-1 block text-sm font-medium text-text">
                Email
              </span>
              <input
                className="w-full rounded-md border border-border px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
                type="email"
                value={inviteEmail}
                onChange={(event) => setInviteEmail(event.target.value)}
                placeholder="user@example.com"
                required
              />
            </label>

            <label className="block">
              <span className="mb-1 block text-sm font-medium text-text">
                Роль
              </span>
              <select
                className="w-full rounded-md border border-border bg-white px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15"
                value={inviteRole}
                onChange={(event) => setInviteRole(event.target.value as Role)}
              >
                {roles.map((role) => (
                  <option key={role} value={role}>
                    {roleLabels[role]}
                  </option>
                ))}
              </select>
            </label>

            <button
              className="self-end rounded-md bg-primary px-4 py-2 font-medium text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-65"
              type="submit"
              disabled={isCreatingInvite}
            >
              {isCreatingInvite ? "Створення..." : "Створити запрошення"}
            </button>
          </form>

          {createdInvite ? (
            <div className="mt-4 rounded-md border border-green-200 bg-green-50 p-3 text-sm text-text">
              <p className="font-medium">
                Запрошення для {createdInvite.email} створено.
              </p>
              <div className="mt-2 flex flex-col gap-2 md:flex-row md:items-center">
                <input
                  className="w-full rounded-md border border-green-200 bg-white px-3 py-2 text-sm"
                  value={inviteUrl}
                  readOnly
                />
                <button
                  className="rounded-md border border-green-300 bg-white px-3 py-2 font-medium text-success transition hover:bg-green-100"
                  type="button"
                  onClick={handleCopyInvite}
                >
                  {copied ? "Скопійовано" : "Копіювати"}
                </button>
              </div>
              <p className="mt-2 text-xs text-muted">
                Посилання одноразове та діє до {formatDate(createdInvite.expires_at)}.
              </p>
            </div>
          ) : null}
        </div>

        {error ? (
          <p className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">
            {error}
          </p>
        ) : null}

        <div className="overflow-hidden rounded-lg border border-border bg-white shadow-sm">
          <div className="overflow-x-auto">
            <table className="min-w-full border-collapse text-left text-sm">
              <thead className="bg-surface text-xs uppercase text-muted">
                <tr>
                  <th className="px-4 py-3 font-semibold">Користувач</th>
                  <th className="px-4 py-3 font-semibold">Email</th>
                  <th className="px-4 py-3 font-semibold">Роль</th>
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
                ) : users.length === 0 ? (
                  <tr>
                    <td className="px-4 py-6 text-muted" colSpan={4}>
                      Користувачів не знайдено.
                    </td>
                  </tr>
                ) : (
                  users.map((user) => (
                    <tr key={user.id}>
                      <td className="px-4 py-3">
                        <p className="font-medium text-text">
                          {`${user.first_name} ${user.last_name}`.trim() || "-"}
                        </p>
                        <p className="text-xs text-muted">{user.id}</p>
                      </td>
                      <td className="px-4 py-3 text-text">{user.email}</td>
                      <td className="px-4 py-3">
                        <select
                          className="rounded-md border border-border bg-white px-3 py-2 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/15 disabled:opacity-65"
                          value={user.role}
                          disabled={updatingUserId === user.id}
                          onChange={(event) =>
                            handleRoleChange(user.id, event.target.value as Role)
                          }
                        >
                          {roles.map((role) => (
                            <option key={role} value={role}>
                              {roleLabels[role]}
                            </option>
                          ))}
                        </select>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-muted">
                        {formatDate(user.created_at)}
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

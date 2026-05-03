"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { getUsers, updateUserRole } from "@/lib/api";
import { getStoredUser, getToken } from "@/lib/auth";
import type { Role, User } from "@/lib/types";

const roles: Role[] = ["client", "operator", "engineer", "admin"];

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

export default function UsersPage() {
  const router = useRouter();
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
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
        setError(err instanceof Error ? err.message : "Не вдалося отримати користувачів.");
      } finally {
        setIsLoading(false);
      }
    }

    void loadUsers();
  }, [router]);

  async function handleRoleChange(userId: string, role: Role) {
    setError(null);
    setUpdatingUserId(userId);

    try {
      const updated = await updateUserRole(userId, role);
      setUsers((current) =>
        current.map((user) => (user.id === updated.id ? updated : user)),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не вдалося оновити роль.");
    } finally {
      setUpdatingUserId(null);
    }
  }

  if (!currentUser) {
    return null;
  }

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
                              {role}
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

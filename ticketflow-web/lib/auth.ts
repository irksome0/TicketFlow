"use client";

import type { AuthResponse, User } from "./types";

const TOKEN_KEY = "ticketflow_token";
const USER_KEY = "ticketflow_user";

function canUseStorage(): boolean {
  return typeof window !== "undefined";
}

export function getToken(): string | null {
  if (!canUseStorage()) {
    return null;
  }
  return window.localStorage.getItem(TOKEN_KEY);
}

export function getStoredUser(): User | null {
  if (!canUseStorage()) {
    return null;
  }

  const raw = window.localStorage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as User;
  } catch {
    clearAuth();
    return null;
  }
}

export function saveAuth(auth: AuthResponse): void {
  if (!canUseStorage()) {
    return;
  }
  window.localStorage.setItem(TOKEN_KEY, auth.token);
  window.localStorage.setItem(USER_KEY, JSON.stringify(auth.user));
}

export function clearAuth(): void {
  if (!canUseStorage()) {
    return;
  }
  window.localStorage.removeItem(TOKEN_KEY);
  window.localStorage.removeItem(USER_KEY);
}

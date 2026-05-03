"use client";

import { clearAuth, getToken } from "./auth";
import type { AuthResponse, ListResponse, Ticket } from "./types";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1";

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
};

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, ...init } = options;
  const token = getToken();
  const headers = new Headers(init.headers);
  const isFormData = body instanceof FormData;
  const requestBody = isFormData ? body : body ? JSON.stringify(body) : undefined;

  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (body && !isFormData) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_URL}${path}`, {
    ...init,
    headers,
    body: requestBody,
  });

  if (response.status === 401) {
    clearAuth();
  }

  if (!response.ok) {
    let message = "Request failed";
    try {
      const data = (await response.json()) as { error?: string };
      message = data.error ?? message;
    } catch {
      message = response.statusText || message;
    }
    throw new Error(message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export function login(email: string, password: string): Promise<AuthResponse> {
  return request<AuthResponse>("/auth/login", {
    method: "POST",
    body: { email, password },
  });
}

export function getTickets(): Promise<ListResponse<Ticket>> {
  return request<ListResponse<Ticket>>("/tickets");
}

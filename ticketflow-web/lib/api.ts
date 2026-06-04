"use client";

import { clearAuth, getToken } from "./auth";
import type {
  Attachment,
  AcceptInviteInput,
  AuthResponse,
  CreateInviteInput,
  Invite,
  ListResponse,
  PublicInvite,
  RegisterOrganizationInput,
  Role,
  Ticket,
  TicketComment,
  TicketPriority,
  TicketStatus,
  User,
} from "./types";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1";

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

function isPublicAuthPath(path: string): boolean {
  return (
    path === "/auth/login" ||
    path === "/auth/register" ||
    path === "/auth/register-organization" ||
    path.startsWith("/users/invites/")
  );
}

function handleUnauthorized(path: string): void {
  clearAuth();

  if (typeof window === "undefined" || isPublicAuthPath(path)) {
    return;
  }

  const currentPath = window.location.pathname + window.location.search;
  const loginPath = `/login?next=${encodeURIComponent(currentPath)}`;

  if (window.location.pathname !== "/login") {
    window.location.replace(loginPath);
  }
}

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
    handleUnauthorized(path);
  }

  if (!response.ok) {
    let message = "Request failed";
    try {
      const data = (await response.json()) as { error?: string };
      message = data.error ?? message;
    } catch {
      message = response.statusText || message;
    }
    throw new ApiError(message, response.status);
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

export function registerOrganization(
  input: RegisterOrganizationInput,
): Promise<AuthResponse> {
  return request<AuthResponse>("/auth/register-organization", {
    method: "POST",
    body: input,
  });
}

export function getTickets(filters?: {
  status?: TicketStatus;
  priority?: TicketPriority;
  assigneeId?: string;
}): Promise<ListResponse<Ticket>> {
  const params = new URLSearchParams();

  if (filters?.status) {
    params.set("status", filters.status);
  }
  if (filters?.priority) {
    params.set("priority", filters.priority);
  }
  if (filters?.assigneeId) {
    params.set("assignee_id", filters.assigneeId);
  }

  const query = params.toString();
  return request<ListResponse<Ticket>>(`/tickets${query ? `?${query}` : ""}`);
}

export function createTicket(input: {
  title: string;
  description: string;
  priority: TicketPriority;
}): Promise<Ticket> {
  return request<Ticket>("/tickets", {
    method: "POST",
    body: input,
  });
}

export function getTicket(id: string): Promise<Ticket> {
  return request<Ticket>(`/tickets/${id}`);
}

export function updateTicketStatus(
  id: string,
  status: TicketStatus,
): Promise<Ticket> {
  return request<Ticket>(`/tickets/${id}/status`, {
    method: "PATCH",
    body: { status },
  });
}

export function assignTicket(
  id: string,
  assigneeId: string | null,
): Promise<Ticket> {
  return request<Ticket>(`/tickets/${id}/assignee`, {
    method: "PATCH",
    body: { assignee_id: assigneeId },
  });
}

export function getTicketComments(ticketId: string): Promise<TicketComment[]> {
  return request<TicketComment[]>(`/tickets/${ticketId}/comments`);
}

export function createTicketComment(
  ticketId: string,
  message: string,
): Promise<TicketComment> {
  return request<TicketComment>(`/tickets/${ticketId}/comments`, {
    method: "POST",
    body: { message },
  });
}

export function getAttachments(ticketId: string): Promise<Attachment[]> {
  return request<Attachment[]>(`/tickets/${ticketId}/attachments`);
}

export function uploadAttachment(
  ticketId: string,
  file: File,
): Promise<Attachment> {
  const formData = new FormData();
  formData.set("file", file);

  return request<Attachment>(`/tickets/${ticketId}/attachments`, {
    method: "POST",
    body: formData,
  });
}

export async function downloadAttachment(
  ticketId: string,
  attachment: Attachment,
): Promise<void> {
  const token = getToken();
  const response = await fetch(
    `${API_URL}/tickets/${ticketId}/attachments/${attachment.id}/download`,
    {
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    },
  );

  if (response.status === 401) {
    handleUnauthorized(`/tickets/${ticketId}/attachments/${attachment.id}/download`);
  }

  if (!response.ok) {
    let message = "Не вдалося завантажити файл";
    try {
      const data = (await response.json()) as { error?: string };
      message = data.error ?? message;
    } catch {
      message = response.statusText || message;
    }
    throw new ApiError(message, response.status);
  }

  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = attachment.file_name;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}

export function getUsers(): Promise<ListResponse<User>> {
  return request<ListResponse<User>>("/users");
}

export function getEngineers(): Promise<ListResponse<User>> {
  return request<ListResponse<User>>("/users/engineers");
}

export function updateUserRole(id: string, role: Role): Promise<User> {
  return request<User>(`/users/${id}/role`, {
    method: "PATCH",
    body: { role },
  });
}

export function createInvite(input: CreateInviteInput): Promise<Invite> {
  return request<Invite>("/users/invites", {
    method: "POST",
    body: input,
  });
}

export function getInvite(token: string): Promise<PublicInvite> {
  return request<PublicInvite>(`/users/invites/${token}`);
}

export function acceptInvite(
  token: string,
  input: AcceptInviteInput,
): Promise<User> {
  return request<User>(`/users/invites/${token}/accept`, {
    method: "POST",
    body: input,
  });
}

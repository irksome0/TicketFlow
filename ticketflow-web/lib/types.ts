export type Role = "client" | "operator" | "engineer" | "admin";

export type TicketStatus =
  | "New"
  | "In Progress"
  | "Resolved"
  | "Closed"
  | "Reopened";

export type TicketPriority = "High" | "Medium" | "Low";

export interface User {
  id: string;
  organization_id: string;
  email: string;
  role: Role;
  first_name: string;
  last_name: string;
  created_at?: string;
  updated_at?: string;
}

export interface Ticket {
  id: string;
  organization_id: string;
  creator_id: string;
  assignee_id: string | null;
  title: string;
  description: string;
  status: TicketStatus;
  priority: TicketPriority;
  created_at: string;
  resolved_at: string | null;
  updated_at: string;
}

export interface Attachment {
  id: string;
  ticket_id: string;
  uploader_id: string;
  file_name: string;
  file_size: number;
  content_type: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface ListResponse<T> {
  data: T[];
  total: number;
}

export interface ApiErrorResponse {
  error: string;
}

import type { Lesson, LessonPayload, Student, User } from "./types";

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body && !(options.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`/api${path}`, {
    credentials: "include",
    cache: "no-store",
    ...options,
    headers,
  });

  if (!response.ok) {
    let message = "请求失败，请稍后再试";
    try {
      const payload = await response.json();
      if (payload?.error) message = payload.error;
    } catch {
      // Keep the generic message for non-JSON responses.
    }
    throw new ApiError(message, response.status);
  }

  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export interface AuthStatus {
  initialized: boolean;
  authenticated: boolean;
  user?: User;
}

export const api = {
  authStatus: () => request<AuthStatus>("/auth/status"),
  setup: (payload: {
    name: string;
    password: string;
  }) =>
    request<{ user: User }>("/auth/setup", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  login: (payload: { name: string; password: string }) =>
    request<{ user: User }>("/auth/login", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  logout: () => request<{ ok: boolean }>("/auth/logout", { method: "POST" }),

  students: (search = "") =>
    request<Student[]>(`/students${search ? `?search=${encodeURIComponent(search)}` : ""}`),
  createStudent: (payload: Omit<Student, "id">) =>
    request<Student>("/students", { method: "POST", body: JSON.stringify(payload) }),
  updateStudent: (id: string, payload: Omit<Student, "id">) =>
    request<Student>(`/students/${id}`, { method: "PATCH", body: JSON.stringify(payload) }),
  deleteStudent: (id: string) =>
    request<{ ok: boolean }>(`/students/${id}`, { method: "DELETE" }),

  lessons: (params: {
    month?: string;
    studentId?: string;
    all?: boolean;
  } = {}) => {
    const query = new URLSearchParams();
    if (params.month) query.set("month", params.month);
    if (params.studentId) query.set("studentId", params.studentId);
    if (params.all) query.set("all", "true");
    return request<Lesson[]>(`/lessons?${query.toString()}`);
  },
  createLesson: (payload: LessonPayload) =>
    request<Lesson>("/lessons", { method: "POST", body: JSON.stringify(payload) }),
  updateLesson: (id: string, payload: LessonPayload) =>
    request<Lesson>(`/lessons/${id}`, { method: "PATCH", body: JSON.stringify(payload) }),
  deleteLesson: (id: string) =>
    request<{ ok: boolean }>(`/lessons/${id}`, { method: "DELETE" }),
};

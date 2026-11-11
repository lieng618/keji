export interface User {
  id: string;
  name: string;
}

export interface Student {
  id: string;
  name: string;
}

export interface Lesson {
  id: string;
  studentId: string;
  studentName: string;
  date: string;
  startTime: string;
  durationMinutes: number;
  status: "scheduled" | "attended";
}

export interface LessonPayload {
  studentId: string;
  date: string;
  startTime: string;
  durationMinutes: number;
  status: "scheduled" | "attended";
}

export interface WheelOption {
  value: string | number;
  label: string;
}

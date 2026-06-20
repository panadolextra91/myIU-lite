import { api } from './api';

export interface AssignmentResponse {
  id: number;
  title: string;
  description: string;
  deadline: string;
  accept_late: boolean;
  late_threshold_days: number | null;
  created_at: string;
}

export interface CreateAssignmentRequest {
  title: string;
  description?: string;
  deadline: string;
  accept_late: boolean;
  late_threshold_days?: number;
}

export interface SubmissionResponse {
  id: number;
  version: number;
  original_filename: string;
  is_late: boolean;
  submitted_at: string;
  late_duration: string;
  score: number | null;
  feedback: string | null;
}

export interface NotificationResponse {
  id: number;
  type: string;
  title: string;
  body: string;
  resource_type: string;
  resource_id: number;
  link: string;
  created_at: string;
  read_at: string | null;
}

export const courseworkApi = {
  listAssignments: async (courseId: number, role: 'student' | 'lecturer') => {
    const res = await api.get<{ data: AssignmentResponse[] }>(`/${role}/courses/${courseId}/assignments`);
    return res.data.data;
  },
  createAssignment: async (courseId: number, body: CreateAssignmentRequest) => {
    const res = await api.post<AssignmentResponse>(`/lecturer/courses/${courseId}/assignments`, body);
    return res.data;
  },
  submitAssignment: async (courseId: number, assignmentId: number, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await api.post<{ data: SubmissionResponse }>(`/student/courses/${courseId}/assignments/${assignmentId}/submissions`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return res.data.data;
  },
  getDownloadUrl: async (courseId: number, assignmentId: number, submissionId: number, role: 'student' | 'lecturer') => {
    const res = await api.get<{ url: string }>(`/${role}/courses/${courseId}/assignments/${assignmentId}/submissions/${submissionId}/download-url`);
    return res.data.url;
  },
  gradeSubmission: async (courseId: number, assignmentId: number, submissionId: number, payload: { score: number; feedback?: string }) => {
    const res = await api.post(`/lecturer/courses/${courseId}/assignments/${assignmentId}/submissions/${submissionId}/grade`, payload);
    return res.data;
  },
  listNotifications: async () => {
    const res = await api.get<{ data: NotificationResponse[] }>('/notifications');
    return res.data.data;
  },
  unreadCount: async () => {
    const res = await api.get<{ count: number }>('/notifications/unread-count');
    return res.data.count;
  },
  markRead: async (id: number) => {
    const res = await api.post(`/notifications/${id}/read`);
    return res.data;
  },
};

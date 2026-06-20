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
};

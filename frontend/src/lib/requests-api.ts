import { api } from './api';

export interface Request {
  id: number;
  course_id: number;
  student_id: number;
  targeted_lecturer_id: number;
  type: string;
  title: string;
  body: string;
  status: 'PENDING' | 'APPROVED' | 'DENIED';
  reply_note?: string;
  created_at: string;
  replied_at?: string;
}

export interface CreateRequestRequest {
  type: 'LEAVE_EARLY' | 'ABSENCE' | 'CUSTOM';
  title: string;
  body: string;
  targeted_lecturer_id: number;
}

export interface ReplyRequestRequest {
  decision: 'APPROVED' | 'DENIED';
  note?: string;
}

export const requestsApi = {
  // Student endpoints
  listCourseLecturers: async (courseId: number) => {
    const response = await api.get<{ data: { lecturer_id: number; username: string; full_name: string }[] }>(`/student/courses/${courseId}/lecturers`);
    return response.data;
  },

  createRequest: async (courseId: number, data: CreateRequestRequest): Promise<Request> => {
    const response = await api.post(`/student/courses/${courseId}/requests`, data);
    return response.data;
  },

  listStudentRequests: async (): Promise<Request[]> => {
    const response = await api.get(`/student/requests`);
    return response.data;
  },

  // Lecturer endpoints
  listLecturerRequests: async (): Promise<Request[]> => {
    const response = await api.get(`/lecturer/requests`);
    return response.data;
  },

  replyRequest: async (requestId: number, data: ReplyRequestRequest): Promise<Request> => {
    const response = await api.post(`/lecturer/requests/${requestId}/reply`, data);
    return response.data;
  },
};

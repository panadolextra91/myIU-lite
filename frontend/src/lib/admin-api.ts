import { api } from './api';

export type UserRole = 'student' | 'lecturer' | 'admin';

export interface UserResponse {
  id: number;
  username: string;
  full_name: string;
  role: UserRole;
  dob: string;
  must_change_password: boolean;
  created_at: string;
}

export interface PaginatedUsers {
  data: UserResponse[];
  total: number;
}

export interface CreateUserRequest {
  id: string;
  full_name: string;
  dob: string;
  role: UserRole;
}

export interface RowError {
  row: number;
  field: string;
  message: string;
}

export const adminApi = {
  listUsers: async (params: { role?: string; search?: string; limit?: number; offset?: number }) => {
    const res = await api.get<PaginatedUsers>('/admin/users', { params });
    return res.data;
  },
  createUser: async (body: CreateUserRequest) => {
    const res = await api.post<{ id: number }>('/admin/users', body);
    return res.data;
  },
  importAccounts: async (role: 'student' | 'lecturer', file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    const endpoint = role === 'student' ? '/admin/students/import' : '/admin/lecturers/import';
    const res = await api.post<{ imported: number }>(endpoint, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return res.data;
  },
  resetPassword: async (id: number) => {
    const res = await api.post<{ status: string }>(`/admin/users/${id}/reset-password`);
    return res.data;
  },
  listCourses: async (params: { term?: string; search?: string; limit?: number; offset?: number }) => {
    const res = await api.get<{ data: CourseResponse[]; total: number }>('/admin/courses', { params });
    return res.data;
  },
  getCourse: async (id: number) => {
    const res = await api.get<CourseResponse>(`/admin/courses/${id}`);
    return res.data;
  },
  createCourse: async (body: CreateCourseRequest) => {
    const res = await api.post<CourseResponse>('/admin/courses', body);
    return res.data;
  },
  updateCourse: async ({ id, ...body }: CreateCourseRequest & { id: number }) => {
    const res = await api.put<CourseResponse>(`/admin/courses/${id}`, body);
    return res.data;
  },
  deleteCourse: async (id: number) => {
    const res = await api.delete<{ status: string }>(`/admin/courses/${id}`);
    return res.data;
  },
  listCourseStudents: async (id: number) => {
    const res = await api.get<{ data: RosterUser[] }>(`/admin/courses/${id}/students`);
    return res.data.data;
  },
  listCourseLecturers: async (id: number) => {
    const res = await api.get<{ data: RosterUser[] }>(`/admin/courses/${id}/lecturers`);
    return res.data.data;
  },
  importStudentsToCourse: async (courseId: number, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await api.post<{ imported: number }>(`/admin/courses/${courseId}/students/import`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return res.data;
  },
  importLecturersToCourse: async (courseId: number, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await api.post<{ imported: number }>(`/admin/courses/${courseId}/lecturers/import`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return res.data;
  },
  removeStudent: async (courseId: number, studentId: number) => {
    const res = await api.delete<{ status: string }>(`/admin/courses/${courseId}/students/${studentId}`);
    return res.data;
  },
  unassignLecturer: async (courseId: number, lecturerId: number) => {
    const res = await api.delete<{ status: string }>(`/admin/courses/${courseId}/lecturers/${lecturerId}`);
    return res.data;
  },
};

export interface CourseResponse {
  id: number;
  code: string;
  name: string;
  term: string;
  start_date: string;
  end_date: string;
  created_at: string;
}

export interface CreateCourseRequest {
  code: string;
  name: string;
  term: string;
  start_date: string;
  end_date: string;
}

export interface RosterUser {
  id: number;
  username: string;
  full_name: string;
}

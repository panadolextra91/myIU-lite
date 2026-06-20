import { api } from './api';

export interface Announcement {
  id: number;
  course_id: number;
  author_id: number;
  title: string;
  body: string;
  audience_type: 'ALL_STUDENTS' | 'SPECIFIC_STUDENTS';
  created_at: string;
}

export interface CreateAnnouncementRequest {
  title: string;
  body: string;
  audience_type: 'ALL_STUDENTS' | 'SPECIFIC_STUDENTS';
  student_ids?: number[];
}

export const announcementsApi = {
  createAnnouncement: async (courseId: number, data: CreateAnnouncementRequest): Promise<Announcement> => {
    const response = await api.post(`/api/lecturer/courses/${courseId}/announcements`, data);
    return response.data;
  },

  listCourseAnnouncements: async (courseId: number): Promise<Announcement[]> => {
    const response = await api.get(`/api/lecturer/courses/${courseId}/announcements`);
    return response.data;
  },

  listCourseStudents: async (courseId: number) => {
    const response = await api.get<{ data: { student_id: number; username: string; full_name: string }[] }>(`/api/lecturer/courses/${courseId}/students`);
    return response.data;
  },

  // Student endpoints
  listStudentAnnouncements: async (courseId: number): Promise<Announcement[]> => {
    const response = await api.get(`/api/student/courses/${courseId}/announcements`);
    return response.data;
  },

  getAnnouncement: async (announcementId: number, courseId: number): Promise<Announcement> => {
    const response = await api.get(`/api/student/announcements/${announcementId}?course_id=${courseId}`);
    return response.data;
  },
};

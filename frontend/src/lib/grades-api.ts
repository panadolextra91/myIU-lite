import { api } from './api';

export interface ComponentResponse {
  id: number;
  parent_id: number | null;
  name: string;
  weight: number;
  source_type: string | null;
  auto_kind: string | null;
}

export interface SchemeResponse {
  id: number;
  course_id: number;
  components: ComponentResponse[];
}

export interface ComponentInput {
  name: string;
  weight: number;
  parent_index?: number;
  source_type?: string;
  auto_kind?: string;
}

export interface SchemeRequest {
  components: ComponentInput[];
}

export interface ScoreEntryRequest {
  student_id: number;
  score: number;
}

export interface ComputedComponent {
  component_id: number;
  score: number;
}

export interface OverallResponse {
  student_id: number;
  overall: number;
  components: ComputedComponent[];
}

export const gradesApi = {
  createScheme: async (courseId: number, data: SchemeRequest): Promise<SchemeResponse> => {
    const res = await api.post(`/api/lecturer/courses/${courseId}/grade-scheme`, data);
    return res.data;
  },

  getScheme: async (courseId: number): Promise<SchemeResponse> => {
    const res = await api.get(`/api/lecturer/courses/${courseId}/grade-scheme`);
    return res.data;
  },

  deleteScheme: async (courseId: number): Promise<void> => {
    await api.delete(`/api/lecturer/courses/${courseId}/grade-scheme`);
  },

  enterScore: async (courseId: number, componentId: number, data: ScoreEntryRequest): Promise<void> => {
    await api.put(`/api/lecturer/courses/${courseId}/grade-components/${componentId}/scores`, data);
  },

  importScoresCSV: async (courseId: number, componentId: number, file: File): Promise<void> => {
    const formData = new FormData();
    formData.append('file', file);
    await api.post(`/api/lecturer/courses/${courseId}/grade-components/${componentId}/scores/import`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },

  getCourseGrades: async (courseId: number): Promise<OverallResponse[]> => {
    const res = await api.get(`/api/lecturer/courses/${courseId}/grades`);
    return res.data;
  },
};

import { api } from './api';

export interface AssignmentResponse {
  id: number;
  title: string;
  description: string;
  deadline: string;
  accept_late: boolean;
  late_threshold_days: number | null;
  max_score: number;
  grading_finalized_at: string | null;
  created_at: string;
}

export interface CreateAssignmentRequest {
  title: string;
  description?: string;
  deadline: string;
  accept_late: boolean;
  late_threshold_days?: number;
  max_score: number;
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

export interface QuizResponse {
  id: number;
  title: string;
  pool_size: number;
  max_questions: number;
  max_grade: number;
  shuffle: boolean;
  retake_count: number;
  open_at: string | null;
  close_at: string | null;
  created_at: string;
}

export interface CreateQuizRequest {
  title: string;
  pool_size: number;
  max_questions: number;
  max_grade: number;
  shuffle: boolean;
  retake_count: number;
  open_at?: string;
  close_at?: string;
}

export interface UIOptionRequest {
  text: string;
  is_correct: boolean;
}

export interface UIQuestionRequest {
  prompt: string;
  question_type: 'single' | 'multi';
  options: UIOptionRequest[];
}

export interface StudentOptionView {
  id: number;
  text: string;
}

export interface StudentQuestionView {
  id: number;
  prompt: string;
  question_type: 'single' | 'multi';
  options: StudentOptionView[];
}

export interface StudentQuizAttemptView {
  id: number;
  quiz_id: number;
  attempt_number: number;
  status: 'IN_PROGRESS' | 'SUBMITTED' | 'AUTO_SUBMITTED';
  score: number | null;
  started_at: string;
  submitted_at: string | null;
  questions: StudentQuestionView[];
  selected_options: Record<number, number[]>;
  correct_options?: Record<number, number[]>;
}

export interface SubmitAttemptResponse {
  score: number;
  official_score: number;
  status: string;
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
  listSubmissions: async (courseId: number, assignmentId: number) => {
    const res = await api.get<{ data: SubmissionResponse[] }>(`/student/courses/${courseId}/assignments/${assignmentId}/submissions`);
    return res.data.data;
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
  finalizeAssignmentGrading: async (courseId: number, assignmentId: number) => {
    const res = await api.post<AssignmentResponse>(`/lecturer/courses/${courseId}/assignments/${assignmentId}/finalize`);
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
  markRead: async (notificationId: number) => {
    const res = await api.post<{ status: string }>(`/notifications/${notificationId}/read`);
    return res.data;
  },
  listQuizzes: async (courseId: number) => {
    const res = await api.get<{ data: QuizResponse[] }>(`/lecturer/courses/${courseId}/quizzes`);
    return res.data.data;
  },
  createQuiz: async (courseId: number, req: CreateQuizRequest) => {
    const res = await api.post<{ data: QuizResponse }>(`/lecturer/courses/${courseId}/quizzes`, req);
    return res.data.data;
  },
  importQuizCSV: async (courseId: number, quizId: number, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await api.post<{ status: string }>(
      `/lecturer/courses/${courseId}/quizzes/${quizId}/questions/import`,
      formData,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    );
    return res.data;
  },
  addUIQuestion: async (courseId: number, quizId: number, req: UIQuestionRequest) => {
    const res = await api.post<{ status: string }>(`/lecturer/courses/${courseId}/quizzes/${quizId}/questions`, req);
    return res.data;
  },
  listStudentQuizzes: async (courseId: number) => {
    const res = await api.get<{ data: QuizResponse[] }>(`/student/courses/${courseId}/quizzes`);
    return res.data.data;
  },
  startAttempt: async (courseId: number, quizId: number) => {
    const res = await api.post<{ data: StudentQuizAttemptView }>(`/student/courses/${courseId}/quizzes/${quizId}/attempts`);
    return res.data.data;
  },
  getAttempt: async (courseId: number, quizId: number, attemptId: number) => {
    const res = await api.get<{ data: StudentQuizAttemptView }>(`/student/courses/${courseId}/quizzes/${quizId}/attempts/${attemptId}`);
    return res.data.data;
  },
  submitAttempt: async (courseId: number, quizId: number, attemptId: number, answers: Record<number, number[]>) => {
    const res = await api.post<{ data: SubmitAttemptResponse }>(`/student/courses/${courseId}/quizzes/${quizId}/attempts/${attemptId}/submit`, { answers });
    return res.data.data;
  },
};

import { createBrowserRouter, Navigate } from 'react-router';
import Login from '@/pages/Login';
import ChangePassword from '@/pages/ChangePassword';
import StudentIndex from '@/pages/student/Index';
import LecturerIndex from '@/pages/lecturer/Index';
import AdminIndex from '@/pages/admin/Index';
import Accounts from '@/pages/admin/Accounts';
import Courses from '@/pages/admin/Courses';
import CourseDetail from '@/pages/admin/CourseDetail';
import Enrollment from '@/pages/admin/Enrollment';
import LecturerAssignment from '@/pages/admin/LecturerAssignment';
import AuditLogs from '@/pages/admin/AuditLogs';
import LecturerAssignments from '@/pages/lecturer/Assignments';
import LecturerQuizzes from '@/pages/lecturer/Quizzes';
import StudentAssignments from '@/pages/student/Assignments';
import StudentQuizzes from '@/pages/student/Quizzes';
import Notifications from '@/pages/Notifications';

import ProtectedRoute from '@/routes/ProtectedRoute';
import RoleGuard from '@/routes/RoleGuard';
import AppLayout from '@/components/AppLayout';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <Navigate to="/login" replace />,
  },
  {
    path: '/login',
    element: <Login />,
  },
  {
    path: '/change-password',
    element: <ProtectedRoute />,
    children: [
      {
        index: true,
        element: <ChangePassword />,
      },
    ],
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppLayout />,
        children: [
          {
            path: '/notifications',
            element: <Notifications />,
          },
          {
            path: '/student',
            element: <RoleGuard allowedRoles={['student']} />,
            children: [
              { index: true, element: <StudentIndex /> },
              { path: 'assignments', element: <StudentAssignments /> },
              { path: 'quizzes', element: <StudentQuizzes /> },
            ],
          },
          {
            path: '/lecturer',
            element: <RoleGuard allowedRoles={['lecturer']} />,
            children: [
              { index: true, element: <LecturerIndex /> },
              { path: 'assignments', element: <LecturerAssignments /> },
              { path: 'courses/:id/quizzes', element: <LecturerQuizzes /> },
            ],
          },
          {
            path: '/admin',
            element: <RoleGuard allowedRoles={['admin']} />,
            children: [
              { index: true, element: <AdminIndex /> },
              { path: 'accounts', element: <Accounts /> },
              { path: 'courses', element: <Courses /> },
              { path: 'courses/:id', element: <CourseDetail /> },
              { path: 'enrollment', element: <Enrollment /> },
              { path: 'lecturers', element: <LecturerAssignment /> },
              { path: 'audit', element: <AuditLogs /> },
            ],
          },
        ],
      },
    ],
  },
]);

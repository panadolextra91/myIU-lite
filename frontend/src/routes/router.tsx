import { createBrowserRouter, Navigate } from 'react-router';
import Login from '@/pages/Login';
import ChangePassword from '@/pages/ChangePassword';
import StudentIndex from '@/pages/student/Index';
import LecturerIndex from '@/pages/lecturer/Index';
import AdminIndex from '@/pages/admin/Index';

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
            path: '/student',
            element: <RoleGuard allowedRoles={['student']} />,
            children: [
              { index: true, element: <StudentIndex /> },
            ],
          },
          {
            path: '/lecturer',
            element: <RoleGuard allowedRoles={['lecturer']} />,
            children: [
              { index: true, element: <LecturerIndex /> },
            ],
          },
          {
            path: '/admin',
            element: <RoleGuard allowedRoles={['admin']} />,
            children: [
              { index: true, element: <AdminIndex /> },
            ],
          },
        ],
      },
    ],
  },
]);

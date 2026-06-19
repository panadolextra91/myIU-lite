import { Navigate, Outlet, useLocation } from 'react-router';
import { useAuthStore } from '@/stores/auth';

export default function ProtectedRoute() {
  const user = useAuthStore((state) => state.user);
  const location = useLocation();

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  if (user.mustChangePassword && location.pathname !== '/change-password') {
    return <Navigate to="/change-password" replace />;
  }

  return <Outlet />;
}

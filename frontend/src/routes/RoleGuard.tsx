import { Navigate, Outlet } from 'react-router';
import { useAuthStore } from '@/stores/auth';

export default function RoleGuard({ allowedRoles }: { allowedRoles: string[] }) {
  const user = useAuthStore((state) => state.user);

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  if (!allowedRoles.includes(user.role)) {
    return <Navigate to={`/${user.role}`} replace />;
  }

  return <Outlet />;
}

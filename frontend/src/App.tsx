import { RouterProvider } from 'react-router';
import { router } from './routes/router';
import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

function App() {
  const [isInitializing, setIsInitializing] = useState(true);
  const setUser = useAuthStore((state) => state.setUser);
  const clearAuth = useAuthStore((state) => state.clear);

  useEffect(() => {
    api.get('/auth/me')
      .then((res) => {
        setUser({
          id: res.data.id,
          username: res.data.username,
          role: res.data.role,
          mustChangePassword: res.data.must_change_password,
        });
      })
      .catch(() => {
        clearAuth();
      })
      .finally(() => {
        setIsInitializing(false);
      });
  }, [setUser, clearAuth]);

  if (isInitializing) {
    return <div className="flex min-h-screen items-center justify-center">Loading...</div>;
  }

  return <RouterProvider router={router} />;
}

export default App;

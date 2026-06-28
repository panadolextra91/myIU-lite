import { Outlet } from 'react-router';
import { useAuthStore } from '@/stores/auth';
import { Button } from '@/components/ui/button';
import { LogOut } from 'lucide-react';
import { NotificationBell } from '@/components/NotificationBell';
import { api } from '@/lib/api';
import { SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar';
import { AdminSidebar } from '@/components/admin/AdminSidebar';
import { ModeToggle } from '@/components/mode-toggle';

export default function AppLayout() {
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clear);

  const handleLogout = async () => {
    try {
      await api.post('/auth/logout');
    } finally {
      clearAuth();
      window.location.assign('/login');
    }
  };

  const content = (
    <div className="flex-1 min-h-screen bg-background flex flex-col w-full">
      <header className="sticky top-0 z-10 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container mx-auto flex h-14 items-center justify-between px-4">
          <div className="flex items-center gap-2">
            {user?.role === 'admin' && <SidebarTrigger />}
            <div className="font-heading text-lg tracking-tight">myIU Lite</div>
          </div>
          {user && (
            <div className="flex items-center gap-4">
              <span className="text-sm text-muted-foreground hidden sm:inline-block">
                {user.username} ({user.role})
              </span>
              <ModeToggle />
              <NotificationBell />
              <Button variant="ghost" size="sm" onClick={handleLogout}>
                <LogOut className="w-4 h-4 mr-2" />
                Logout
              </Button>
            </div>
          )}
        </div>
      </header>
      <main className="flex-1 container mx-auto p-4 max-w-full">
        <Outlet />
      </main>
    </div>
  );

  if (user?.role === 'admin') {
    return (
      <SidebarProvider>
        <AdminSidebar />
        {content}
      </SidebarProvider>
    );
  }

  return content;
}

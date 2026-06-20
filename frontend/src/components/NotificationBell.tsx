import { useQuery } from '@tanstack/react-query';
import { Bell } from 'lucide-react';
import { Link } from 'react-router';
import { courseworkApi } from '@/lib/coursework-api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

export function NotificationBell() {
  const { data: count = 0 } = useQuery({
    queryKey: ['notifications', 'unreadCount'],
    queryFn: courseworkApi.unreadCount,
    refetchInterval: 30000, // poll every 30s
  });

  return (
    <Link to="/notifications">
      <Button variant="ghost" size="icon" className="relative">
        <Bell className="h-5 w-5" />
        {count > 0 && (
          <Badge 
            variant="destructive" 
            className="absolute -top-1 -right-1 h-5 w-5 flex items-center justify-center p-0 text-[10px]"
          >
            {count > 99 ? '99+' : count}
          </Badge>
        )}
      </Button>
    </Link>
  );
}

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { courseworkApi } from '@/lib/coursework-api';
import { formatDistanceToNow } from 'date-fns';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';

export default function Notifications() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: notifications, isLoading } = useQuery({
    queryKey: ['notifications', 'list'],
    queryFn: courseworkApi.listNotifications,
  });

  const markReadMutation = useMutation({
    mutationFn: courseworkApi.markRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleClick = (notif: any) => {
    if (!notif.read_at) {
      markReadMutation.mutate(notif.id);
    }
    if (notif.link) {
      navigate(notif.link);
    }
  };

  if (isLoading) {
    return (
      <div className="max-w-3xl mx-auto p-4 space-y-4">
        <h1 className="text-2xl font-bold mb-6">Notifications</h1>
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-24 w-full" />
        ))}
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto p-4">
      <h1 className="text-2xl font-bold mb-6">Notifications</h1>
      <div className="space-y-4">
        {notifications?.length === 0 ? (
          <p className="text-muted-foreground text-center py-8">No notifications yet.</p>
        ) : (
          notifications?.map((notif) => (
            <Card 
              key={notif.id} 
              className={`cursor-pointer transition-colors hover:bg-muted/50 ${!notif.read_at ? 'border-primary/50 bg-primary/5' : ''}`}
              onClick={() => handleClick(notif)}
            >
              <CardHeader className="py-3">
                <div className="flex justify-between items-start">
                  <CardTitle className="text-base flex items-center gap-2">
                    {!notif.read_at && <span className="w-2 h-2 rounded-full bg-primary" />}
                    {notif.title}
                  </CardTitle>
                  <span className="text-xs text-muted-foreground whitespace-nowrap ml-4">
                    {formatDistanceToNow(new Date(notif.created_at), { addSuffix: true })}
                  </span>
                </div>
              </CardHeader>
              <CardContent className="py-3 pt-0 text-sm text-muted-foreground">
                {notif.body}
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
}

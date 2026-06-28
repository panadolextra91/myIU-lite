import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { courseworkApi } from '@/lib/coursework-api';
import { formatDistanceToNow } from 'date-fns';
import { Card } from '@/components/ui/card';
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
      <div className="max-w-3xl mx-auto px-4 md:px-0 py-10">
        <div className="mb-10">
          <h1 className="text-3xl font-normal tracking-tight text-foreground">Notifications</h1>
          <div className="h-px w-16 bg-border mt-2" />
        </div>
        <div className="flex flex-col gap-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-28 w-full rounded-none" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto px-4 md:px-0 py-10">
      <div className="mb-10">
        <h1 className="text-3xl font-normal tracking-tight text-foreground">Notifications</h1>
        <div className="h-px w-16 bg-border mt-2" />
      </div>

      <div className="flex flex-col gap-4">
        {notifications?.length === 0 ? (
          <p className="text-muted-foreground text-center py-8">No notifications yet.</p>
        ) : (
          notifications?.map((notif) => {
            const unread = !notif.read_at;
            return (
              <Card
                key={notif.id}
                onClick={() => handleClick(notif)}
                className={`cursor-pointer rounded-none shadow-none p-6 transition-colors duration-200 hover:border-muted-foreground ${
                  unread ? '' : 'opacity-90'
                }`}
              >
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center min-w-0">
                    {unread ? (
                      <span className="w-2 h-2 rounded-full bg-primary mr-3 shrink-0" />
                    ) : (
                      <div className="w-5 shrink-0" />
                    )}
                    <h2
                      className={`text-lg truncate ${
                        unread ? 'text-foreground font-medium' : 'text-foreground opacity-80'
                      }`}
                    >
                      {notif.title}
                    </h2>
                  </div>
                  <time className="font-mono text-sm text-muted-foreground opacity-70 whitespace-nowrap ml-4 shrink-0">
                    {formatDistanceToNow(new Date(notif.created_at), { addSuffix: true })}
                  </time>
                </div>
                <p
                  className={`text-muted-foreground leading-relaxed ${
                    unread ? '' : 'opacity-70'
                  }`}
                >
                  {notif.body}
                </p>
              </Card>
            );
          })
        )}
      </div>
    </div>
  );
}

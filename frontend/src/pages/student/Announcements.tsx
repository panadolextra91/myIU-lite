import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { formatDistanceToNow } from 'date-fns';

import { announcementsApi } from '@/lib/announcements-api';
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';

export default function StudentAnnouncements() {
  const { id, announcementId } = useParams<{ id: string; announcementId?: string }>();
  const courseId = parseInt(id || '0', 10);

  const { data: announcements, isLoading } = useQuery({
    queryKey: ['student-announcements', courseId],
    queryFn: () => announcementsApi.listStudentAnnouncements(courseId),
  });

  useEffect(() => {
    if (announcementId && announcements?.length) {
      const el = document.getElementById(`announcement-${announcementId}`);
      if (el) el.scrollIntoView({ behavior: 'smooth' });
    }
  }, [announcementId, announcements]);

  return (
    <div className="mx-auto max-w-4xl p-8">
      <header className="mb-12 border-b pb-6">
        <h1 className="mb-2 text-3xl font-normal tracking-tight text-primary">Announcements</h1>
        <p className="text-base italic text-muted-foreground">
          View all announcements for this course.
        </p>
      </header>

      {isLoading ? (
        <div className="flex flex-col gap-6">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="[--card-spacing:--spacing(6)]">
              <CardHeader>
                <Skeleton className="h-7 w-1/3" />
                <CardAction>
                  <Skeleton className="h-4 w-20" />
                </CardAction>
              </CardHeader>
              <CardContent>
                <Skeleton className="h-4 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : announcements?.length === 0 ? (
        <Card className="[--card-spacing:--spacing(6)]">
          <CardContent className="text-center text-muted-foreground">
            No announcements found.
          </CardContent>
        </Card>
      ) : (
        <div className="flex flex-col gap-6">
          {announcements?.map((a) => (
            <Card
              key={a.id}
              id={`announcement-${a.id}`}
              className="[--card-spacing:--spacing(6)] transition-transform duration-300 hover:-translate-y-0.5"
            >
              <CardHeader>
                <CardTitle className="text-2xl text-primary">{a.title}</CardTitle>
                <CardAction className="self-baseline font-mono text-sm tabular-nums text-muted-foreground opacity-70">
                  {formatDistanceToNow(new Date(a.created_at), { addSuffix: true })}
                </CardAction>
              </CardHeader>
              <CardContent>
                <p className="max-w-3xl whitespace-pre-wrap text-base leading-relaxed text-foreground">
                  {a.body}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { formatDistanceToNow } from 'date-fns';

import { announcementsApi } from '@/lib/announcements-api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
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
    <div className="max-w-4xl mx-auto p-8 space-y-8">
      <div>
        <h1 className="text-3xl font-bold mb-2">Announcements</h1>
        <p className="text-muted-foreground">View all announcements for this course.</p>
      </div>

      <div>
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <Card key={i}>
                <CardHeader><Skeleton className="h-6 w-1/3" /></CardHeader>
                <CardContent><Skeleton className="h-4 w-full" /></CardContent>
              </Card>
            ))}
          </div>
        ) : announcements?.length === 0 ? (
          <Card>
            <CardContent className="pt-6 text-center text-muted-foreground">
              No announcements found.
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            {announcements?.map((a) => (
              <Card key={a.id} id={`announcement-${a.id}`}>
                <CardHeader>
                  <CardTitle className="flex justify-between items-start">
                    <span>{a.title}</span>
                    <span className="text-sm font-normal text-muted-foreground">
                      {formatDistanceToNow(new Date(a.created_at), { addSuffix: true })}
                    </span>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="whitespace-pre-wrap text-sm">{a.body}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

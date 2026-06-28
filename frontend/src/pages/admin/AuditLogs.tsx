import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { format } from 'date-fns';

type AuditLog = {
  actor_id: number | null;
  action: string;
  target_type: string | null;
  target_id: number | null;
  operation_id: string | null;
  affected_count: number | null;
  metadata: Record<string, unknown>;
  created_at: string;
};

type AuditLogsResponse = {
  data: AuditLog[];
  total: number;
};

export default function AuditLogs() {
  const { data, isLoading } = useQuery({
    queryKey: ['audit-logs'],
    queryFn: async () => {
      const res = await api.get<AuditLogsResponse>('/admin/audit-logs');
      return res.data;
    },
  });

  return (
    <div className="space-y-8">
      <header className="border-b pb-6">
        <h1 className="text-3xl font-normal tracking-tight">Audit Logs</h1>
        <p className="mt-2 text-muted-foreground">
          System-wide audit trail of all administrative actions.
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle className="text-xl font-normal tracking-tight">Recent Activity</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Timestamp</TableHead>
                    <TableHead>Actor ID</TableHead>
                    <TableHead>Action</TableHead>
                    <TableHead>Target</TableHead>
                    <TableHead className="text-right">Affected</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data?.data.map((log, idx) => (
                    <TableRow key={idx}>
                      <TableCell className="font-mono tabular-nums text-muted-foreground">
                        {format(new Date(log.created_at), 'yyyy-MM-dd HH:mm:ss')}
                      </TableCell>
                      <TableCell className="font-mono tabular-nums">{log.actor_id ?? 'SYSTEM'}</TableCell>
                      <TableCell className="font-mono tabular-nums text-xs">{log.action}</TableCell>
                      <TableCell className="font-mono tabular-nums text-muted-foreground">
                        {log.target_type} {log.target_id ? `#${log.target_id}` : ''}
                      </TableCell>
                      <TableCell className="text-right font-mono tabular-nums">{log.affected_count}</TableCell>
                    </TableRow>
                  ))}
                  {data?.data.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} className="py-10 text-center text-muted-foreground">
                        No logs found.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

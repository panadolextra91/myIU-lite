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
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Audit Logs</h2>
        <p className="text-muted-foreground">System-wide audit trail of all administrative actions.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Recent Activity</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <p>Loading...</p>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Timestamp</TableHead>
                    <TableHead>Actor ID</TableHead>
                    <TableHead>Action</TableHead>
                    <TableHead>Target</TableHead>
                    <TableHead>Affected</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data?.data.map((log, idx) => (
                    <TableRow key={idx}>
                      <TableCell>{format(new Date(log.created_at), 'yyyy-MM-dd HH:mm:ss')}</TableCell>
                      <TableCell>{log.actor_id ?? 'SYSTEM'}</TableCell>
                      <TableCell className="font-mono text-xs">{log.action}</TableCell>
                      <TableCell>
                        {log.target_type} {log.target_id ? `#${log.target_id}` : ''}
                      </TableCell>
                      <TableCell>{log.affected_count}</TableCell>
                    </TableRow>
                  ))}
                  {data?.data.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center">No logs found.</TableCell>
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

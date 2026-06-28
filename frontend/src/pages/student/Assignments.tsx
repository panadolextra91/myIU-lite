import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { courseworkApi } from '@/lib/coursework-api';
import type { SubmissionResponse } from '@/lib/coursework-api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Download, Clock } from 'lucide-react';
import { toast } from 'sonner';

export default function StudentAssignments() {
  const [courseId, setCourseId] = useState<number>(1);
  const [files, setFiles] = useState<Record<number, File | null>>({});
  const [submissions, setSubmissions] = useState<Record<number, SubmissionResponse>>({});

  const { data: assignments } = useQuery({
    queryKey: ['assignments', courseId],
    queryFn: () => courseworkApi.listAssignments(courseId, 'student'),
  });

  useQuery({
    queryKey: ['assignments-submissions', courseId, assignments?.map(a => a.id)],
    queryFn: async () => {
      if (!assignments) return {};
      const results: Record<number, SubmissionResponse> = {};
      await Promise.all(
        assignments.map(async (a) => {
          const subs = await courseworkApi.listSubmissions(courseId, a.id);
          if (subs.length > 0) {
            results[a.id] = subs[0]; // Active submission is the first one
          }
        })
      );
      setSubmissions(results);
      return results;
    },
    enabled: !!assignments && assignments.length > 0,
  });

  const submitMutation = useMutation({
    mutationFn: ({ assignmentId, file }: { assignmentId: number; file: File }) =>
      courseworkApi.submitAssignment(courseId, assignmentId, file),
    onSuccess: (data, variables) => {
      toast.success('Assignment submitted');
      setSubmissions(prev => ({ ...prev, [variables.assignmentId]: data }));
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: { message?: string } } } };
      toast.error(error.response?.data?.error?.message || 'Failed to submit assignment');
    },
  });

  const handleDownload = async (assignmentId: number, submissionId: number) => {
    try {
      const url = await courseworkApi.getDownloadUrl(courseId, assignmentId, submissionId, 'student');
      window.open(url, '_blank');
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: { message?: string } } } };
      toast.error(error.response?.data?.error?.message || 'Failed to get download URL');
    }
  };

  return (
    <div className="max-w-6xl mx-auto space-y-12 py-2">
      <div className="flex items-end justify-between">
        <h1 className="text-4xl font-normal tracking-tight">Assignments</h1>
        <div className="w-24">
          <Label
            htmlFor="course-id"
            className="mb-2 block text-xs font-medium uppercase tracking-wider text-muted-foreground"
          >
            Course ID
          </Label>
          <Input
            id="course-id"
            type="number"
            value={courseId}
            onChange={(e) => setCourseId(Number(e.target.value))}
            className="font-mono tabular-nums"
            placeholder="000"
          />
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Title</TableHead>
            <TableHead>Deadline</TableHead>
            <TableHead>Status / Submissions</TableHead>
            <TableHead>Submit</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {assignments?.map((a) => {
            const sub = submissions[a.id];
            return (
              <TableRow key={a.id}>
                <TableCell className="py-6">
                  <span className="font-heading text-2xl tracking-tight text-foreground">{a.title}</span>
                </TableCell>
                <TableCell className="py-6">
                  <span className="font-mono tabular-nums text-muted-foreground">{new Date(a.deadline).toLocaleString()}</span>
                </TableCell>
                <TableCell className="py-6">
                  {sub ? (
                    <div className="flex flex-col gap-1">
                      <Button
                        variant="link"
                        className="h-auto w-fit gap-2 p-0"
                        onClick={() => handleDownload(a.id, sub.id)}
                      >
                        <Download className="h-4 w-4" strokeWidth={1.5} />
                        Download Submission <span className="font-mono">v{sub.version}</span>
                      </Button>
                      {sub.is_late && (
                        <span className="flex items-center gap-1 font-mono text-xs text-destructive">
                          <Clock className="h-3.5 w-3.5" strokeWidth={1.5} />
                          Late: {sub.late_duration}
                        </span>
                      )}
                    </div>
                  ) : (
                    <span className="italic text-muted-foreground">No submission this session</span>
                  )}
                </TableCell>
                <TableCell className="py-6">
                  <div className="flex max-w-[240px] flex-col gap-3">
                    <Input
                      type="file"
                      accept=".pdf,.zip"
                      onChange={(e) => setFiles(prev => ({ ...prev, [a.id]: e.target.files?.[0] || null }))}
                    />
                    <Button
                      disabled={!files[a.id] || submitMutation.isPending}
                      onClick={() => submitMutation.mutate({ assignmentId: a.id, file: files[a.id]! })}
                    >
                      Submit
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            );
          })}
          {assignments?.length === 0 && (
            <TableRow>
              <TableCell colSpan={4} className="py-10 text-center italic text-muted-foreground">No assignments found</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}

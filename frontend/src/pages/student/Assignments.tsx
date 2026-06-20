import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { courseworkApi } from '@/lib/coursework-api';
import type { SubmissionResponse } from '@/lib/coursework-api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
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
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Assignments</h1>
        <Input 
          type="number" 
          value={courseId} 
          onChange={(e) => setCourseId(Number(e.target.value))} 
          className="w-24"
          placeholder="Course ID"
        />
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
                <TableCell>{a.title}</TableCell>
                <TableCell>{new Date(a.deadline).toLocaleString()}</TableCell>
                <TableCell>
                  {sub ? (
                    <div className="flex flex-col space-y-1">
                      <Button variant="link" className="p-0 h-auto w-fit" onClick={() => handleDownload(a.id, sub.id)}>
                        Download Submission v{sub.version}
                      </Button>
                      {sub.is_late && <span className="text-red-500 text-sm">Late: {sub.late_duration}</span>}
                    </div>
                  ) : (
                    <span className="text-muted-foreground">No submission this session</span>
                  )}
                </TableCell>
                <TableCell className="flex items-center space-x-2">
                  <Input 
                    type="file" 
                    accept=".pdf,.zip" 
                    onChange={(e) => setFiles(prev => ({ ...prev, [a.id]: e.target.files?.[0] || null }))} 
                    className="max-w-[200px]"
                  />
                  <Button 
                    disabled={!files[a.id] || submitMutation.isPending}
                    onClick={() => submitMutation.mutate({ assignmentId: a.id, file: files[a.id]! })}
                  >
                    Submit
                  </Button>
                </TableCell>
              </TableRow>
            );
          })}
          {assignments?.length === 0 && (
            <TableRow>
              <TableCell colSpan={4} className="text-center text-muted-foreground">No assignments found</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}

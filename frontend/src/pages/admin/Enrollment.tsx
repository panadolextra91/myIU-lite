import { useState, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import { toast } from 'sonner';
import { Info, Upload } from 'lucide-react';

import { adminApi } from '@/lib/admin-api';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';

export default function Enrollment() {
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [courseId, setCourseId] = useState<string>('');
  const [errors, setErrors] = useState<Array<{ row: number; field: string; message: string }>>([]);

  const { data: coursesData, isLoading } = useQuery({
    queryKey: ['courses', '', '', 0],
    queryFn: () => adminApi.listCourses({ limit: 1000 }),
  });

  const importMutation = useMutation({
    mutationFn: ({ id, file }: { id: number; file: File }) => adminApi.importStudentsToCourse(id, file),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['course-students'] });
      toast.success(`Imported ${data.imported} students successfully!`);
      setErrors([]);
      if (fileInputRef.current) fileInputRef.current.value = '';
    },
    onError: (err) => {
      if (isAxiosError(err) && err.response?.status === 422) {
        setErrors(err.response.data.errors || []);
        toast.error('Validation failed. See table for details.');
      } else if (isAxiosError(err) && err.response?.data?.error?.message) {
        toast.error(err.response.data.error.message);
      } else {
        toast.error('Failed to import students');
      }
      if (fileInputRef.current) fileInputRef.current.value = '';
    },
  });

  const onImportFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!courseId) {
      toast.error('Please select a course first');
      return;
    }
    const file = e.target.files?.[0];
    if (file) {
      importMutation.mutate({ id: parseInt(courseId, 10), file });
    }
  };

  return (
    <div className="max-w-3xl space-y-12">
      <div>
        <h1 className="mb-2 text-3xl tracking-tight">Student Enrollment</h1>
        <p className="text-muted-foreground">Enroll students into active courses via CSV upload.</p>
      </div>

      <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
        <div className="flex flex-col gap-2">
          <label className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
            Target Course
          </label>
          {isLoading ? (
            <Skeleton className="h-11 w-[280px]" />
          ) : (
            <Select value={courseId} onValueChange={setCourseId}>
              <SelectTrigger className="h-11 w-[280px]">
                <SelectValue placeholder="Select a course..." />
              </SelectTrigger>
              <SelectContent>
                {coursesData?.data.map(c => (
                  <SelectItem key={c.id} value={c.id.toString()}>{c.code} - {c.name} ({c.term})</SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>

        <label className="cursor-pointer">
          <Button render={<span />} variant="outline" className="h-11 gap-2 px-6" disabled={importMutation.isPending || !courseId}>
            <Upload strokeWidth={1.5} />
            Import Student CSV
          </Button>
          <input ref={fileInputRef} type="file" className="hidden" accept=".csv" onChange={onImportFile} disabled={!courseId} />
        </label>
      </div>

      <section className="rounded-lg border bg-card p-6">
        <div className="mb-4 flex items-center gap-3">
          <Info className="text-gold" strokeWidth={1.5} />
          <h2 className="text-xl tracking-tight">CSV Format Requirements</h2>
        </div>
        <ul className="space-y-3">
          <li className="flex items-start">
            <span className="mr-3 text-gold">•</span>
            <span className="text-muted-foreground">Must contain a column named <code className="rounded bg-muted px-1 font-mono">student_id</code> in the header.</span>
          </li>
          <li className="flex items-start">
            <span className="mr-3 text-gold">•</span>
            <span className="text-muted-foreground">Each row must have a valid, active student ID (e.g., <code className="rounded bg-muted px-1 font-mono">ITITIU19000</code>).</span>
          </li>
          <li className="flex items-start">
            <span className="mr-3 text-gold">•</span>
            <span className="text-muted-foreground">File will be rejected if ANY row contains an invalid ID or if there are duplicates in the file.</span>
          </li>
          <li className="flex items-start">
            <span className="mr-3 text-gold">•</span>
            <span className="text-muted-foreground">Already enrolled students in the file will be silently skipped (idempotent).</span>
          </li>
        </ul>
      </section>

      {errors.length > 0 && (
        <div className="space-y-4">
          <div className="border-l-4 border-destructive py-1 pl-4 font-medium text-destructive">
            Import failed. Please fix the following errors and try again. No students were enrolled.
          </div>
          <div className="rounded-md border border-destructive/50">
            <Table>
              <TableHeader className="bg-destructive/10">
                <TableRow>
                  <TableHead className="text-destructive">Row</TableHead>
                  <TableHead className="text-destructive">Field</TableHead>
                  <TableHead className="text-destructive">Error Message</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {errors.map((e, i) => (
                  <TableRow key={i}>
                    <TableCell className="font-mono font-medium tabular-nums text-destructive">{e.row}</TableCell>
                    <TableCell className="text-destructive">{e.field}</TableCell>
                    <TableCell className="text-destructive">{e.message}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}
    </div>
  );
}

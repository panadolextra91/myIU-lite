import { useState, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import { toast } from 'sonner';
import { Info, CircleCheck, CircleAlert, RefreshCw } from 'lucide-react';

import { adminApi } from '@/lib/admin-api';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';

export default function LecturerAssignment() {
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [courseId, setCourseId] = useState<string>('');
  const [errors, setErrors] = useState<Array<{ row: number; field: string; message: string }>>([]);

  const { data: coursesData, isLoading } = useQuery({
    queryKey: ['courses', '', '', 0],
    queryFn: () => adminApi.listCourses({ limit: 1000 }),
  });

  const importMutation = useMutation({
    mutationFn: ({ id, file }: { id: number; file: File }) => adminApi.importLecturersToCourse(id, file),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['course-lecturers'] });
      toast.success(`Assigned ${data.imported} lecturers successfully!`);
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
        toast.error('Failed to assign lecturers');
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
      <header className="space-y-2">
        <h1 className="text-3xl font-normal tracking-tight text-foreground">Lecturer Assignment</h1>
        <p className="text-muted-foreground">Assign lecturers to active courses via CSV upload.</p>
      </header>

      <section className="flex flex-wrap items-center gap-6">
        {isLoading ? (
          <Skeleton className="h-12 w-[280px]" />
        ) : (
          <Select value={courseId} onValueChange={setCourseId}>
            <SelectTrigger className="h-12 w-[280px]">
              <SelectValue placeholder="Select a course..." />
            </SelectTrigger>
            <SelectContent>
              {coursesData?.data.map(c => (
                <SelectItem key={c.id} value={c.id.toString()}>{c.code} - {c.name} ({c.term})</SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}

        <label className="cursor-pointer">
          <Button
            variant="outline"
            render={<span />}
            className="h-12 px-8"
            disabled={importMutation.isPending || !courseId}
          >
            Import Lecturer CSV
          </Button>
          <input ref={fileInputRef} type="file" className="hidden" accept=".csv" onChange={onImportFile} disabled={!courseId} />
        </label>
      </section>

      <section className="rounded-lg border bg-card p-6 sm:p-8">
        <h3 className="mb-6 text-2xl font-normal tracking-tight text-foreground">CSV Format Requirements</h3>
        <ul className="list-none space-y-4 p-0 text-muted-foreground">
          <li className="flex gap-4">
            <Info className="mt-0.5 size-5 shrink-0 text-gold" strokeWidth={1.5} />
            <span>
              Must contain a column named{' '}
              <code className="rounded border bg-background px-1.5 py-0.5 font-mono text-sm">lecturer_id</code>{' '}
              in the header.
            </span>
          </li>
          <li className="flex gap-4">
            <CircleCheck className="mt-0.5 size-5 shrink-0 text-gold" strokeWidth={1.5} />
            <span>Each row must have a valid, active lecturer ID.</span>
          </li>
          <li className="flex gap-4">
            <CircleAlert className="mt-0.5 size-5 shrink-0 text-gold" strokeWidth={1.5} />
            <span>File will be rejected if ANY row contains an invalid ID or if there are duplicates in the file.</span>
          </li>
          <li className="flex gap-4">
            <RefreshCw className="mt-0.5 size-5 shrink-0 text-gold" strokeWidth={1.5} />
            <span>Already assigned lecturers in the file will be silently skipped (idempotent).</span>
          </li>
        </ul>
      </section>

      {errors.length > 0 && (
        <section className="space-y-4">
          <div className="border-l-4 border-destructive py-1 pl-4 font-medium text-destructive">
            Import failed. Please fix the following errors and try again. No lecturers were assigned.
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
        </section>
      )}
    </div>
  );
}

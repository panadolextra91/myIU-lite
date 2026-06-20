import { useState, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import { toast } from 'sonner';

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
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Student Enrollment</h2>
        <p className="text-muted-foreground">Enroll students into active courses via CSV upload.</p>
      </div>

      <div className="flex flex-col sm:flex-row gap-4 sm:items-center">
        {isLoading ? (
          <Skeleton className="h-10 w-[280px]" />
        ) : (
          <Select value={courseId} onValueChange={setCourseId}>
            <SelectTrigger className="w-[280px]">
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
          <Button render={<span />} disabled={importMutation.isPending || !courseId}>
            Import Student CSV
          </Button>
          <input ref={fileInputRef} type="file" className="hidden" accept=".csv" onChange={onImportFile} disabled={!courseId} />
        </label>
      </div>

      <div className="rounded-md border p-4 bg-muted/50">
        <h3 className="font-semibold mb-2">CSV Format Requirements</h3>
        <ul className="list-disc pl-5 text-sm text-muted-foreground space-y-1">
          <li>Must contain a column named <code>student_id</code> in the header.</li>
          <li>Each row must have a valid, active student ID (e.g., <code>ITITIU19000</code>).</li>
          <li>File will be rejected if ANY row contains an invalid ID or if there are duplicates in the file.</li>
          <li>Already enrolled students in the file will be silently skipped (idempotent).</li>
        </ul>
      </div>

      {errors.length > 0 && (
        <div className="space-y-4">
          <div className="text-destructive font-medium border-l-4 border-destructive pl-4 py-1">
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
                    <TableCell className="font-medium text-destructive">{e.row}</TableCell>
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

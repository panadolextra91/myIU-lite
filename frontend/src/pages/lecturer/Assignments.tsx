import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { courseworkApi } from '@/lib/coursework-api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { toast } from 'sonner';

const schema = z.object({
  title: z.string().min(1, 'Title is required'),
  description: z.string().optional(),
  deadline: z.string().min(1, 'Deadline is required'),
  accept_late: z.boolean(),
  late_threshold_days: z.coerce.number().optional(),
});

type FormValues = z.infer<typeof schema>;

export default function LecturerAssignments() {
  const [courseId, setCourseId] = useState<number>(1);
  const [open, setOpen] = useState(false);
  const queryClient = useQueryClient();

  const { data: assignments } = useQuery({
    queryKey: ['assignments', courseId],
    queryFn: () => courseworkApi.listAssignments(courseId, 'lecturer'),
  });

  const form = useForm<FormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(schema) as any,
    defaultValues: { accept_late: false, title: '', description: '', deadline: '' },
  });

  const mutation = useMutation({
    mutationFn: (values: FormValues) => {
      const payload = {
        ...values,
        deadline: new Date(values.deadline).toISOString(),
      };
      return courseworkApi.createAssignment(courseId, payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['assignments', courseId] });
      toast.success('Assignment created');
      setOpen(false);
      form.reset();
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: { message?: string } } } };
      toast.error(error.response?.data?.error?.message || 'Failed to create assignment');
    },
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  const handleDownload = async (assignmentId: number, submissionId: number) => {
    try {
      const url = await courseworkApi.getDownloadUrl(courseId, assignmentId, submissionId, 'lecturer');
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
        <div className="flex items-center space-x-2">
          <Input 
            type="number" 
            value={courseId} 
            onChange={(e) => setCourseId(Number(e.target.value))} 
            className="w-24"
            placeholder="Course ID"
          />
          <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger render={<Button />}>
              Create Assignment
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>New Assignment</DialogTitle>
              </DialogHeader>
              <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                  <FormField
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    control={form.control as any}
                    name="title"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Title</FormLabel>
                        <FormControl>
                          <Input {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    control={form.control as any}
                    name="description"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Description</FormLabel>
                        <FormControl>
                          <Input {...field} value={field.value || ''} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    control={form.control as any}
                    name="deadline"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Deadline</FormLabel>
                        <FormControl>
                          <Input type="datetime-local" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    control={form.control as any}
                    name="accept_late"
                    render={({ field }) => (
                      <FormItem className="flex items-center space-x-2">
                        <FormControl>
                          <input type="checkbox" checked={field.value} onChange={field.onChange} />
                        </FormControl>
                        <FormLabel className="!mt-0">Accept Late Submissions</FormLabel>
                      </FormItem>
                    )}
                  />
                  <FormField
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    control={form.control as any}
                    name="late_threshold_days"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Late Threshold (Days)</FormLabel>
                        <FormControl>
                          <Input type="number" {...field} value={field.value || ''} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <Button type="submit" disabled={mutation.isPending}>Save</Button>
                </form>
              </Form>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>ID</TableHead>
            <TableHead>Title</TableHead>
            <TableHead>Deadline</TableHead>
            <TableHead>Accept Late</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {assignments?.map((a) => (
            <TableRow key={a.id}>
              <TableCell>
                <Button variant="link" className="p-0 h-auto" onClick={() => handleDownload(a.id, 1)}>
                  {a.id}
                </Button>
              </TableCell>
              <TableCell>{a.title}</TableCell>
              <TableCell>{new Date(a.deadline).toLocaleString()}</TableCell>
              <TableCell>{a.accept_late ? `Yes (${a.late_threshold_days || 'unlimited'} days)` : 'No'}</TableCell>
            </TableRow>
          ))}
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

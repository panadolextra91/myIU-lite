import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { isAxiosError } from 'axios';
import { toast } from 'sonner';
import { Link } from 'react-router';
import { Search, Calendar } from 'lucide-react';

import { adminApi } from '@/lib/admin-api';
import type { CourseResponse } from '@/lib/admin-api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';

const courseSchema = z.object({
  code: z.string().min(1, 'Code is required'),
  name: z.string().min(1, 'Name is required'),
  term: z.string().min(1, 'Term is required'),
  start_date: z.string().min(1, 'Start date is required'),
  end_date: z.string().min(1, 'End date is required'),
}).refine(data => {
  return new Date(data.end_date) >= new Date(data.start_date);
}, { message: "End date cannot be before start date", path: ["end_date"] });

export default function Courses() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [termFilter, setTermFilter] = useState('');
  const [page, setPage] = useState(0);
  const pageSize = 50;

  const [createOpen, setCreateOpen] = useState(false);
  const [editCourse, setEditCourse] = useState<CourseResponse | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['courses', search, termFilter, page],
    queryFn: () => adminApi.listCourses({
      search: search || undefined,
      term: termFilter || undefined,
      limit: pageSize,
      offset: page * pageSize,
    }),
  });

  const form = useForm<z.infer<typeof courseSchema>>({
    resolver: zodResolver(courseSchema),
    defaultValues: { code: '', name: '', term: '', start_date: '', end_date: '' },
  });

  const createMutation = useMutation({
    mutationFn: adminApi.createCourse,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['courses'] });
      toast.success('Course created successfully');
      setCreateOpen(false);
      form.reset();
    },
    onError: (err) => {
      if (isAxiosError(err) && err.response?.data?.error?.message) {
        toast.error(err.response.data.error.message);
      } else {
        toast.error('Failed to create course');
      }
    },
  });

  const updateMutation = useMutation({
    mutationFn: adminApi.updateCourse,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['courses'] });
      toast.success('Course updated successfully');
      setEditCourse(null);
    },
    onError: (err) => {
      if (isAxiosError(err) && err.response?.data?.error?.message) {
        toast.error(err.response.data.error.message);
      } else {
        toast.error('Failed to update course');
      }
    },
  });

  const deleteMutation = useMutation({
    mutationFn: adminApi.deleteCourse,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['courses'] });
      toast.success('Course soft-deleted successfully');
    },
    onError: () => {
      toast.error('Failed to delete course');
    },
  });

  const handleEdit = (course: CourseResponse) => {
    setEditCourse(course);
    form.reset({
      code: course.code,
      name: course.name,
      term: course.term,
      start_date: course.start_date,
      end_date: course.end_date,
    });
  };

  const onSubmit = (v: z.infer<typeof courseSchema>) => {
    if (editCourse) {
      updateMutation.mutate({ ...v, id: editCourse.id });
    } else {
      createMutation.mutate(v);
    }
  };

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end justify-between">
        <div className="space-y-1">
          <h1 className="text-3xl font-normal tracking-tight">Courses</h1>
          <p className="text-muted-foreground">Manage courses and lifecycles.</p>
        </div>

        <Dialog open={createOpen || !!editCourse} onOpenChange={(open) => {
          if (!open) {
            setCreateOpen(false);
            setEditCourse(null);
            form.reset({ code: '', name: '', term: '', start_date: '', end_date: '' });
          } else {
            setCreateOpen(true);
          }
        }}>
          <DialogTrigger render={<Button />}>
            Create Course
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{editCourse ? 'Edit Course' : 'Create Course'}</DialogTitle>
            </DialogHeader>
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                  control={form.control}
                  name="code"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Course Code</FormLabel>
                      <FormControl><Input placeholder="CS101" {...field} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Course Name</FormLabel>
                      <FormControl><Input placeholder="Intro to Computer Science" {...field} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="term"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Term</FormLabel>
                      <FormControl><Input placeholder="Spring 2026" {...field} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <div className="grid grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="start_date"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Start Date</FormLabel>
                        <FormControl><Input type="date" {...field} /></FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="end_date"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>End Date</FormLabel>
                        <FormControl><Input type="date" {...field} /></FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending} className="w-full">
                  {editCourse ? 'Update' : 'Create'}
                </Button>
              </form>
            </Form>
          </DialogContent>
        </Dialog>
      </div>

      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative w-full sm:max-w-sm sm:flex-grow">
          <Search
            strokeWidth={1.5}
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
          />
          <Input
            placeholder="Search code or name..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-10"
          />
        </div>
        <div className="relative w-full sm:w-48">
          <Calendar
            strokeWidth={1.5}
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
          />
          <Input
            placeholder="Filter by term..."
            value={termFilter}
            onChange={(e) => setTermFilter(e.target.value)}
            className="pl-10"
          />
        </div>
      </div>

      <div className="overflow-x-auto rounded-md border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Code</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Term</TableHead>
              <TableHead>Start Date</TableHead>
              <TableHead>End Date</TableHead>
              <TableHead className="w-[140px]">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={6} className="text-center text-muted-foreground">Loading...</TableCell></TableRow>
            ) : data?.data.length === 0 ? (
              <TableRow><TableCell colSpan={6} className="text-center text-muted-foreground">No courses found.</TableCell></TableRow>
            ) : (
              data?.data.map((c) => (
                <TableRow key={c.id}>
                  <TableCell>
                    <Button
                      render={<Link to={`/admin/courses/${c.id}`} />}
                      variant="link"
                      className="px-0 font-mono font-medium underline underline-offset-4"
                    >
                      {c.code}
                    </Button>
                  </TableCell>
                  <TableCell>{c.name}</TableCell>
                  <TableCell>{c.term}</TableCell>
                  <TableCell className="font-mono tabular-nums text-muted-foreground">{c.start_date}</TableCell>
                  <TableCell className="font-mono tabular-nums text-muted-foreground">{c.end_date}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Button variant="ghost" size="sm" onClick={() => handleEdit(c)}>Edit</Button>
                      <AlertDialog>
                        <AlertDialogTrigger render={<Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" />}>
                          Delete
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>Soft-delete Course?</AlertDialogTitle>
                            <AlertDialogDescription>
                              This course ({c.code}) will be hidden from lists, but history is preserved.
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                            <AlertDialogAction 
                              onClick={() => deleteMutation.mutate(c.id)}
                              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                            >
                              Soft-delete
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      
      {data && data.total > pageSize && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Showing <span className="font-mono tabular-nums">{page * pageSize + 1}</span> to <span className="font-mono tabular-nums">{Math.min((page + 1) * pageSize, data.total)}</span> of <span className="font-mono tabular-nums">{data.total}</span>
          </p>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" disabled={page === 0} onClick={() => setPage(p => p - 1)}>Previous</Button>
            <Button variant="outline" size="sm" disabled={(page + 1) * pageSize >= data.total} onClick={() => setPage(p => p + 1)}>Next</Button>
          </div>
        </div>
      )}
    </div>
  );
}

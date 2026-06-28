import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { formatDistanceToNow } from 'date-fns';
import { Info, History } from 'lucide-react';

import { announcementsApi } from '@/lib/announcements-api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Skeleton } from '@/components/ui/skeleton';
import { Textarea } from '@/components/ui/textarea';

const formSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  body: z.string().min(1, 'Body is required'),
  audience_type: z.enum(['ALL_STUDENTS', 'SPECIFIC_STUDENTS']),
  student_ids: z.array(z.number()).optional(),
}).refine(data => {
  if (data.audience_type === 'SPECIFIC_STUDENTS') {
    return data.student_ids && data.student_ids.length > 0;
  }
  return true;
}, {
  message: 'Must select at least one student for specific audience',
  path: ['student_ids'],
});

const labelCaps = 'text-xs font-medium uppercase tracking-wider text-muted-foreground';

export default function LecturerAnnouncements() {
  const { id } = useParams<{ id: string }>();
  const courseId = parseInt(id || '0', 10);
  const queryClient = useQueryClient();

  const { data: announcements, isLoading: loadingAnnc } = useQuery({
    queryKey: ['lecturer-announcements', courseId],
    queryFn: () => announcementsApi.listCourseAnnouncements(courseId),
  });

  const { data: studentsObj, isLoading: loadingStudents } = useQuery({
    queryKey: ['lecturer-course-students', courseId],
    queryFn: () => announcementsApi.listCourseStudents(courseId),
  });
  const students = studentsObj?.data || [];

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      title: '',
      body: '',
      audience_type: 'ALL_STUDENTS',
      student_ids: [],
    },
  });

  const audienceType = form.watch('audience_type');

  const createMutation = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) => announcementsApi.createAnnouncement(courseId, {
      title: values.title,
      body: values.body,
      audience_type: values.audience_type,
      student_ids: values.audience_type === 'SPECIFIC_STUDENTS' ? values.student_ids : [],
    }),
    onSuccess: () => {
      toast.success('Announcement sent');
      form.reset();
      queryClient.invalidateQueries({ queryKey: ['lecturer-announcements', courseId] });
    },
    onError: (err: unknown) => {
      const e = err as { response?: { data?: { error?: { message?: string } } } };
      toast.error(e.response?.data?.error?.message || 'Failed to send announcement');
    },
  });

  const onSubmit = (values: z.infer<typeof formSchema>) => {
    createMutation.mutate(values);
  };

  return (
    <div className="max-w-6xl mx-auto p-8 space-y-8">
      {/* Page Header */}
      <header className="border-b pb-6">
        <h1 className="text-3xl font-normal tracking-tight">Announcements</h1>
        <p className="text-muted-foreground mt-2">Send announcements and view your history.</p>
      </header>

      {/* Layout Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* Compose Announcement */}
        <Card className="lg:col-span-7">
          <CardHeader>
            <CardTitle className="text-2xl font-normal tracking-tight text-primary">Compose Announcement</CardTitle>
            <CardDescription className="flex items-center gap-2 text-destructive">
              <Info className="h-[18px] w-[18px]" strokeWidth={1.5} />
              Announcements cannot be edited or deleted once sent.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                <FormField
                  control={form.control}
                  name="title"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className={labelCaps}>Title</FormLabel>
                      <FormControl>
                        <Input placeholder="E.g., Midterm Exam Room Update" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="body"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className={labelCaps}>Body</FormLabel>
                      <FormControl>
                        <Textarea
                          className="flex min-h-[120px] w-full"
                          placeholder="Type your announcement here..."
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="audience_type"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className={labelCaps}>Audience</FormLabel>
                      <Select onValueChange={field.onChange} defaultValue={field.value}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Select audience" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="ALL_STUDENTS">All Enrolled Students</SelectItem>
                          <SelectItem value="SPECIFIC_STUDENTS">Specific Students</SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {audienceType === 'SPECIFIC_STUDENTS' && (
                  <FormField
                    control={form.control}
                    name="student_ids"
                    render={() => (
                      <FormItem>
                        <FormLabel className={labelCaps}>Select Students</FormLabel>
                        <div className="border rounded-md p-4 max-h-[300px] overflow-y-auto bg-muted/30 space-y-3">
                          {loadingStudents ? (
                            <Skeleton className="h-6 w-full" />
                          ) : students.length === 0 ? (
                            <div className="text-sm text-muted-foreground">No students enrolled.</div>
                          ) : (
                            students.map((student) => (
                              <FormField
                                key={student.student_id}
                                control={form.control}
                                name="student_ids"
                                render={({ field }) => {
                                  return (
                                    <FormItem
                                      key={student.student_id}
                                      className="flex flex-row items-center gap-3 space-y-0"
                                    >
                                      <FormControl>
                                        <Checkbox
                                          checked={field.value?.includes(student.student_id)}
                                          onCheckedChange={(checked) => {
                                            const current = field.value || [];
                                            return checked
                                              ? field.onChange([...current, student.student_id])
                                              : field.onChange(
                                                  current.filter((val) => val !== student.student_id)
                                                );
                                          }}
                                        />
                                      </FormControl>
                                      <FormLabel className="flex flex-col gap-0.5 font-normal cursor-pointer">
                                        <span className="text-sm text-foreground">{student.full_name}</span>
                                        <span className="font-mono text-xs text-muted-foreground">{student.username}</span>
                                      </FormLabel>
                                    </FormItem>
                                  );
                                }}
                              />
                            ))
                          )}
                        </div>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}

                <Button type="submit" className="w-full" disabled={createMutation.isPending}>
                  {createMutation.isPending ? 'Sending...' : 'Send Announcement'}
                </Button>
              </form>
            </Form>
          </CardContent>
        </Card>

        {/* Sent Announcements */}
        <section className="lg:col-span-5 space-y-6">
          <h2 className="text-2xl font-normal tracking-tight flex items-center gap-3">
            <History className="h-6 w-6 text-primary" strokeWidth={1.5} />
            Sent Announcements
          </h2>
          {loadingAnnc ? (
            <div className="space-y-4">
              {[1, 2].map((i) => (
                <Card key={i}>
                  <CardHeader><Skeleton className="h-6 w-1/3" /></CardHeader>
                  <CardContent><Skeleton className="h-4 w-full" /></CardContent>
                </Card>
              ))}
            </div>
          ) : announcements?.length === 0 ? (
            <p className="text-muted-foreground">You haven't sent any announcements yet.</p>
          ) : (
            <div className="space-y-4">
              {announcements?.map((a) => (
                <Card key={a.id}>
                  <CardContent className="p-6">
                    <div className="flex justify-between items-start gap-3 mb-3">
                      <span className="font-mono text-xs uppercase tracking-widest text-muted-foreground tabular-nums">
                        {formatDistanceToNow(new Date(a.created_at), { addSuffix: true })}
                      </span>
                      <Badge variant="secondary" className="shrink-0">
                        To: {a.audience_type === 'ALL_STUDENTS' ? 'All Students' : 'Specific Students'}
                      </Badge>
                    </div>
                    <h3 className="text-xl font-normal tracking-tight text-primary mb-3">{a.title}</h3>
                    <p className="whitespace-pre-wrap text-sm text-muted-foreground leading-relaxed border-l-2 pl-4">
                      {a.body}
                    </p>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

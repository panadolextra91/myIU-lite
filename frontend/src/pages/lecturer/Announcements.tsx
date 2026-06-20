import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { formatDistanceToNow } from 'date-fns';

import { announcementsApi } from '@/lib/announcements-api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Skeleton } from '@/components/ui/skeleton';

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
    <div className="max-w-4xl mx-auto p-8 space-y-8">
      <div>
        <h1 className="text-3xl font-bold mb-2">Announcements</h1>
        <p className="text-muted-foreground">Send announcements and view your history.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Compose Announcement</CardTitle>
          <CardDescription>Announcements cannot be edited or deleted once sent.</CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <FormField
                control={form.control}
                name="title"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Title</FormLabel>
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
                    <FormLabel>Body</FormLabel>
                    <FormControl>
                      <textarea 
                        className="flex min-h-[120px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                        placeholder="Write your announcement here..."
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
                    <FormLabel>Audience</FormLabel>
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
                      <div className="mb-4">
                        <FormLabel className="text-base">Select Students</FormLabel>
                      </div>
                      <div className="grid grid-cols-2 gap-4 border p-4 rounded-md max-h-[300px] overflow-y-auto">
                        {loadingStudents ? (
                          <Skeleton className="h-6 w-full" />
                        ) : students.length === 0 ? (
                          <div className="text-sm text-muted-foreground col-span-2">No students enrolled.</div>
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
                                    className="flex flex-row items-start space-x-3 space-y-0"
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
                                    <FormLabel className="font-normal">
                                      {student.full_name} ({student.username})
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

              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? 'Sending...' : 'Send Announcement'}
              </Button>
            </form>
          </Form>
        </CardContent>
      </Card>

      <div>
        <h2 className="text-2xl font-bold mb-4">Sent Announcements</h2>
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
                <CardHeader>
                  <CardTitle className="flex justify-between items-start">
                    <span>{a.title}</span>
                    <span className="text-sm font-normal text-muted-foreground">
                      {formatDistanceToNow(new Date(a.created_at), { addSuffix: true })}
                    </span>
                  </CardTitle>
                  <CardDescription>
                    To: {a.audience_type === 'ALL_STUDENTS' ? 'All Students' : 'Specific Students'}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <p className="whitespace-pre-wrap text-sm">{a.body}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

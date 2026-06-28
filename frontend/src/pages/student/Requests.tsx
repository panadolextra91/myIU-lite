import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { formatDistanceToNow } from 'date-fns';

import { requestsApi } from '@/lib/requests-api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';

const formSchema = z.object({
  type: z.enum(['LEAVE_EARLY', 'ABSENCE', 'CUSTOM']),
  title: z.string().min(1, 'Title is required'),
  body: z.string().min(1, 'Body is required'),
  targeted_lecturer_id: z.coerce.number().min(1, 'Please select a lecturer'),
});

const labelCaps = 'text-xs font-medium uppercase tracking-wider text-muted-foreground';

export default function StudentRequests() {
  const { id } = useParams<{ id: string }>();
  const courseId = parseInt(id || '0', 10);
  const queryClient = useQueryClient();

  const { data: requests, isLoading: loadingReqs } = useQuery({
    queryKey: ['student-requests'],
    queryFn: () => requestsApi.listStudentRequests(),
  });

  const { data: lecturersObj, isLoading: loadingLecturers } = useQuery({
    queryKey: ['student-course-lecturers', courseId],
    queryFn: () => requestsApi.listCourseLecturers(courseId),
    enabled: courseId > 0,
  });
  const lecturers = lecturersObj?.data || [];

  const form = useForm({
    resolver: zodResolver(formSchema),
    defaultValues: {
      type: 'LEAVE_EARLY',
      title: '',
      body: '',
    },
  });

  const createMutation = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) => requestsApi.createRequest(courseId, {
      type: values.type,
      title: values.title,
      body: values.body,
      targeted_lecturer_id: values.targeted_lecturer_id,
    }),
    onSuccess: () => {
      toast.success('Request sent successfully');
      form.reset({
        type: 'LEAVE_EARLY',
        title: '',
        body: '',
      });
      queryClient.invalidateQueries({ queryKey: ['student-requests'] });
    },
    onError: (err: unknown) => {
      const e = err as { response?: { data?: { error?: { message?: string } } } };
      toast.error(e.response?.data?.error?.message || 'Failed to send request');
    },
  });

  const onSubmit = (values: z.infer<typeof formSchema>) => {
    createMutation.mutate(values);
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'APPROVED': return <Badge variant="success" className="uppercase tracking-wide">Approved</Badge>;
      case 'DENIED': return <Badge variant="destructive" className="uppercase tracking-wide">Denied</Badge>;
      default: return <Badge variant="warning" className="uppercase tracking-wide">Pending</Badge>;
    }
  };

  return (
    <div className="max-w-6xl mx-auto p-8 space-y-10">
      <header>
        <h1 className="text-3xl font-normal tracking-tight">My Requests</h1>
        <p className="text-muted-foreground mt-2">Send a request to your lecturer and view their replies.</p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* Send Request */}
        <section className="lg:col-span-5">
          <Card>
            <CardHeader>
              <CardTitle className="text-2xl font-normal tracking-tight">Send Request</CardTitle>
              <CardDescription>Send an absence request, leave early request, or custom request.</CardDescription>
            </CardHeader>
            <CardContent>
              <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="targeted_lecturer_id"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel className={labelCaps}>Send To</FormLabel>
                          <Select onValueChange={field.onChange} value={field.value?.toString() || ''}>
                            <FormControl>
                              <SelectTrigger className="w-full">
                                <SelectValue placeholder={loadingLecturers ? "Loading..." : "Select lecturer"} />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              {lecturers.map(l => (
                                <SelectItem key={l.lecturer_id} value={l.lecturer_id.toString()}>
                                  {l.full_name} ({l.username})
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="type"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel className={labelCaps}>Request Type</FormLabel>
                          <Select onValueChange={field.onChange} defaultValue={field.value}>
                            <FormControl>
                              <SelectTrigger className="w-full">
                                <SelectValue placeholder="Select type" />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              <SelectItem value="LEAVE_EARLY">Leave Early</SelectItem>
                              <SelectItem value="ABSENCE">Absence</SelectItem>
                              <SelectItem value="CUSTOM">Custom</SelectItem>
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  <FormField
                    control={form.control}
                    name="title"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel className={labelCaps}>Title</FormLabel>
                        <FormControl>
                          <Input placeholder="E.g., Medical Appointment" {...field} />
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
                        <FormLabel className={labelCaps}>Details</FormLabel>
                        <FormControl>
                          <Textarea
                            className="min-h-[120px] resize-none"
                            placeholder="Please provide details for your request..."
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <Button
                    type="submit"
                    disabled={createMutation.isPending}
                    className="w-full h-12 uppercase tracking-widest"
                  >
                    {createMutation.isPending ? 'Sending...' : 'Send Request'}
                  </Button>
                </form>
              </Form>
            </CardContent>
          </Card>
        </section>

        {/* Request History */}
        <section className="lg:col-span-7 space-y-6">
          <h2 className="text-2xl font-normal tracking-tight border-b pb-2">Request History</h2>
          {loadingReqs ? (
            <div className="space-y-4">
              {[1, 2].map((i) => (
                <Card key={i}>
                  <CardHeader><Skeleton className="h-6 w-1/3" /></CardHeader>
                  <CardContent><Skeleton className="h-4 w-full" /></CardContent>
                </Card>
              ))}
            </div>
          ) : requests?.length === 0 ? (
            <p className="text-muted-foreground">You haven't sent any requests.</p>
          ) : (
            <div className="space-y-4">
              {requests?.map((req) => (
                <Card key={req.id} className="transition-all hover:ring-gold/30">
                  <CardContent className="space-y-2">
                    <div className="flex justify-between items-start gap-3">
                      <h3 className="font-heading text-xl font-medium tracking-tight text-foreground">{req.title}</h3>
                      {getStatusBadge(req.status)}
                    </div>

                    <div className="flex flex-wrap gap-x-4 gap-y-1 font-mono text-xs uppercase tracking-wide text-muted-foreground/70">
                      <span>Type: {req.type}</span>
                      <span>Sent {formatDistanceToNow(new Date(req.created_at), { addSuffix: true })}</span>
                    </div>

                    <p className="whitespace-pre-wrap text-sm text-foreground pt-1">{req.body}</p>

                    {req.status !== 'PENDING' && (
                      <div className="mt-2 border-l-2 border-border bg-muted/40 p-4 rounded-r-md">
                        <p className="text-sm italic text-muted-foreground whitespace-pre-wrap">
                          {req.reply_note || <span className="not-italic text-muted-foreground">No additional notes provided.</span>}
                        </p>
                        {req.replied_at && (
                          <p className="mt-2 text-right font-mono text-xs uppercase tracking-wide text-muted-foreground/60">
                            Replied {formatDistanceToNow(new Date(req.replied_at), { addSuffix: true })}
                          </p>
                        )}
                      </div>
                    )}
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

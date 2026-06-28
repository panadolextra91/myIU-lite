import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { formatDistanceToNow } from 'date-fns';
import { Reply, Inbox } from 'lucide-react';

import { requestsApi, type Request as ReqModel } from '@/lib/requests-api';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';

const replySchema = z.object({
  decision: z.enum(['APPROVED', 'DENIED']),
  note: z.string().optional(),
});

export default function LecturerRequestInbox() {
  const queryClient = useQueryClient();
  const [activeDialog, setActiveDialog] = useState<number | null>(null);

  const { data: requests, isLoading } = useQuery({
    queryKey: ['lecturer-requests'],
    queryFn: () => requestsApi.listLecturerRequests(),
  });

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'APPROVED': return <Badge variant="success">Approved</Badge>;
      case 'DENIED': return <Badge variant="destructive">Denied</Badge>;
      default: return <Badge variant="warning">Pending</Badge>;
    }
  };

  return (
    <div className="max-w-5xl mx-auto p-8 space-y-10">
      <header className="border-b border-border pb-8">
        <h1 className="text-4xl font-normal italic tracking-tight text-foreground mb-2">Request Inbox</h1>
        <p className="text-muted-foreground">
          Manage student requests directed to you. Review academic adjustments, extensions, and grade inquiries.
        </p>
      </header>

      <div>
        {isLoading ? (
          <div className="flex flex-col gap-6">
            {[1, 2, 3].map((i) => (
              <Card key={i} className="p-6 flex flex-col gap-4">
                <Skeleton className="h-7 w-1/3" />
                <Skeleton className="h-4 w-2/3" />
                <Skeleton className="h-12 w-full" />
              </Card>
            ))}
          </div>
        ) : requests?.length === 0 ? (
          <Card className="p-12 flex flex-col items-center justify-center text-center gap-3">
            <Inbox className="h-10 w-10 text-muted-foreground" strokeWidth={1.5} />
            <p className="text-muted-foreground">Your inbox is empty.</p>
          </Card>
        ) : (
          <div className="flex flex-col gap-6">
            {requests?.map((req) => (
              <Card key={req.id} className="p-6 flex flex-col gap-4">
                <div className="flex justify-between items-start gap-4">
                  <div className="flex flex-col gap-2">
                    <div className="flex flex-wrap items-center gap-4">
                      <h2 className="text-2xl font-normal tracking-tight text-foreground">{req.title}</h2>
                      {getStatusBadge(req.status)}
                    </div>
                    <div className="flex flex-wrap items-center gap-2 font-mono text-sm text-muted-foreground">
                      <span>Type: <span className="text-foreground">{req.type}</span></span>
                      <span className="text-border">|</span>
                      <span>From Student ID: <span className="text-foreground tabular-nums">{req.student_id}</span></span>
                      <span className="text-border">|</span>
                      <span>Sent <span className="tabular-nums">{formatDistanceToNow(new Date(req.created_at), { addSuffix: true })}</span></span>
                    </div>
                  </div>
                  {req.status === 'PENDING' && (
                    <Dialog
                      open={activeDialog === req.id}
                      onOpenChange={(isOpen) => setActiveDialog(isOpen ? req.id : null)}
                    >
                      <DialogTrigger>
                        <Button size="sm" className="gap-2 shrink-0">
                          <Reply className="h-4 w-4" strokeWidth={1.5} />
                          Reply
                        </Button>
                      </DialogTrigger>
                      <DialogContent>
                        <DialogHeader>
                          <DialogTitle>Reply to Request</DialogTitle>
                        </DialogHeader>
                        <ReplyForm
                          req={req}
                          onSuccess={() => {
                            setActiveDialog(null);
                            queryClient.invalidateQueries({ queryKey: ['lecturer-requests'] });
                          }}
                        />
                      </DialogContent>
                    </Dialog>
                  )}
                </div>

                <blockquote className="border-l-2 border-border pl-6 py-1 italic text-muted-foreground whitespace-pre-wrap max-w-3xl">
                  {req.body}
                </blockquote>

                {req.status !== 'PENDING' && (
                  <div className="mt-2 p-4 bg-muted border border-border rounded-lg">
                    <div className="flex justify-between items-center mb-2">
                      <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Your Reply</span>
                      {req.replied_at && (
                        <span className="text-xs font-mono tabular-nums text-muted-foreground">
                          Replied {formatDistanceToNow(new Date(req.replied_at), { addSuffix: true })}
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-foreground whitespace-pre-wrap">
                      {req.reply_note || <span className="italic text-muted-foreground">No additional notes provided.</span>}
                    </p>
                  </div>
                )}
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function ReplyForm({ req, onSuccess }: { req: ReqModel; onSuccess: () => void }) {
  const form = useForm({
    resolver: zodResolver(replySchema),
    defaultValues: {
      decision: 'APPROVED',
      note: '',
    },
  });

  const replyMutation = useMutation({
    mutationFn: (values: z.infer<typeof replySchema>) => requestsApi.replyRequest(req.id, values),
    onSuccess: () => {
      toast.success('Reply sent successfully');
      onSuccess();
    },
    onError: (err: unknown) => {
      const e = err as { response?: { data?: { error?: { message?: string } } } };
      toast.error(e.response?.data?.error?.message || 'Failed to send reply');
    },
  });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit((v) => replyMutation.mutate(v))} className="space-y-4">
        <FormField
          control={form.control}
          name="decision"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Decision</FormLabel>
              <Select onValueChange={field.onChange} defaultValue={field.value}>
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder="Select decision" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value="APPROVED">Approve</SelectItem>
                  <SelectItem value="DENIED">Deny</SelectItem>
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="note"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Note (Optional)</FormLabel>
              <FormControl>
                <Textarea
                  className="flex min-h-[100px] w-full"
                  placeholder="Explain your decision..."
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit" className="w-full" disabled={replyMutation.isPending}>
          {replyMutation.isPending ? 'Sending...' : 'Send Reply'}
        </Button>
      </form>
    </Form>
  );
}

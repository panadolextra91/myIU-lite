import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { formatDistanceToNow } from 'date-fns';

import { requestsApi, type Request as ReqModel } from '@/lib/requests-api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
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
      case 'APPROVED': return <Badge className="bg-green-600 hover:bg-green-700">Approved</Badge>;
      case 'DENIED': return <Badge variant="destructive">Denied</Badge>;
      default: return <Badge variant="secondary">Pending</Badge>;
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-8 space-y-8">
      <div>
        <h1 className="text-3xl font-bold mb-2">Request Inbox</h1>
        <p className="text-muted-foreground">Manage student requests directed to you.</p>
      </div>

      <div>
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <Card key={i}>
                <CardHeader><Skeleton className="h-6 w-1/3" /></CardHeader>
                <CardContent><Skeleton className="h-4 w-full" /></CardContent>
              </Card>
            ))}
          </div>
        ) : requests?.length === 0 ? (
          <Card>
            <CardContent className="pt-6 text-center text-muted-foreground">
              Your inbox is empty.
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            {requests?.map((req) => (
              <Card key={req.id}>
                <CardHeader className="pb-3">
                  <div className="flex justify-between items-start">
                    <div>
                      <CardTitle className="text-lg flex items-center gap-2">
                        {getStatusBadge(req.status)}
                        {req.title}
                      </CardTitle>
                      <CardDescription className="mt-1">
                        Type: {req.type} | From Student ID: {req.student_id} | Sent {formatDistanceToNow(new Date(req.created_at), { addSuffix: true })}
                      </CardDescription>
                    </div>
                    {req.status === 'PENDING' && (
                      <Dialog 
                        open={activeDialog === req.id} 
                        onOpenChange={(isOpen) => setActiveDialog(isOpen ? req.id : null)}
                      >
                        <DialogTrigger>
                          <Button size="sm">Reply</Button>
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
                </CardHeader>
                <CardContent>
                  <p className="whitespace-pre-wrap text-sm text-foreground mb-4">{req.body}</p>
                  
                  {req.status !== 'PENDING' && (
                    <div className="bg-muted p-4 rounded-md mt-4">
                      <div className="flex justify-between items-center mb-2">
                        <span className="font-medium text-sm">Your Reply</span>
                        {req.replied_at && (
                          <span className="text-xs text-muted-foreground">
                            {formatDistanceToNow(new Date(req.replied_at), { addSuffix: true })}
                          </span>
                        )}
                      </div>
                      <p className="text-sm whitespace-pre-wrap">
                        {req.reply_note || <span className="italic text-muted-foreground">No additional notes provided.</span>}
                      </p>
                    </div>
                  )}
                </CardContent>
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

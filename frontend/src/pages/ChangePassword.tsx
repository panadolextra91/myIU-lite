import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useNavigate } from 'react-router';
import { KeyRound, CircleAlert } from 'lucide-react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { toast } from 'sonner';
import axios from 'axios';

import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';

const formSchema = z.object({
  current_password: z.string().min(1, 'Current password is required'),
  new_password: z.string().min(6, 'Password must be at least 6 characters'),
  confirm_password: z.string().min(1, 'Confirm password is required'),
}).refine((data) => data.new_password === data.confirm_password, {
  message: "Passwords don't match",
  path: ["confirm_password"],
});

export default function ChangePassword() {
  const navigate = useNavigate();
  const clearAuth = useAuthStore((state) => state.clear);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      current_password: '',
      new_password: '',
      confirm_password: '',
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    setError(null);
    try {
      await api.post('/auth/change-password', values);
      clearAuth();
      toast.success("Password changed successfully. Please log in again.");
      navigate('/login');
    } catch (err: unknown) {
      if (axios.isAxiosError(err)) {
        const code = err.response?.data?.error?.code;
        if (code === 'current_password_invalid') {
          setError('Current password is incorrect');
        } else if (code === 'same_as_current') {
          setError('New password must differ from the current one');
        } else if (code === 'password_too_short') {
          setError('Password too short');
        } else if (code === 'confirm_mismatch') {
          setError("Passwords don't match");
        } else {
          setError('An unexpected error occurred');
        }
      } else {
        setError('An unexpected error occurred');
      }
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <div className="w-full max-w-md flex flex-col gap-6 rounded-lg border bg-card p-10">
        <header className="flex flex-col items-center text-center">
          <KeyRound className="size-11 text-primary/80 mb-4" strokeWidth={1.25} />
          <h1 className="font-heading text-3xl tracking-tight">Change Password</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            You must change your password before continuing.
          </p>
        </header>

        {error && (
          <div className="flex items-center justify-center gap-2 rounded-md bg-destructive p-3 text-center text-sm text-destructive-foreground">
            <CircleAlert className="size-4 shrink-0" />
            {error}
          </div>
        )}

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-5">
            <FormField
              control={form.control}
              name="current_password"
              render={({ field }) => (
                <FormItem className="gap-2">
                  <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Current Password
                  </FormLabel>
                  <FormControl>
                    <Input className="h-12" type="password" placeholder="Enter current password" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="new_password"
              render={({ field }) => (
                <FormItem className="gap-2">
                  <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    New Password
                  </FormLabel>
                  <FormControl>
                    <Input className="h-12" type="password" placeholder="Enter new password" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="confirm_password"
              render={({ field }) => (
                <FormItem className="gap-2">
                  <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Confirm New Password
                  </FormLabel>
                  <FormControl>
                    <Input className="h-12" type="password" placeholder="Confirm new password" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" className="mt-1 h-12 w-full">Change Password</Button>
          </form>
        </Form>
      </div>
    </div>
  );
}

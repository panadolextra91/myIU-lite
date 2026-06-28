import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useNavigate } from 'react-router';
import { Lock, CircleAlert } from 'lucide-react';
import axios from 'axios';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

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
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(6, 'Password must be at least 6 characters'),
});

export default function Login() {
  const navigate = useNavigate();
  const setUser = useAuthStore((state) => state.setUser);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: '',
      password: '',
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    setError(null);
    try {
      const res = await api.post('/auth/login', values);
      const user = res.data;
      setUser({
        id: user.id,
        username: user.username,
        role: user.role,
        mustChangePassword: user.must_change_password,
      });

      if (user.must_change_password) {
        navigate('/change-password');
      } else {
        if (user.role === 'student') navigate('/student');
        else if (user.role === 'lecturer') navigate('/lecturer');
        else if (user.role === 'admin') navigate('/admin');
      }
    } catch (err: unknown) {
      if (axios.isAxiosError(err) && err.response?.status === 401) {
        setError('Invalid username or password');
      } else {
        setError('An unexpected error occurred');
      }
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <div className="w-full max-w-md flex flex-col gap-6 rounded-lg border bg-card p-10">
        <header className="flex flex-col items-center text-center">
          <Lock className="size-11 text-primary/80 mb-4" strokeWidth={1.25} />
          <h1 className="font-heading text-3xl tracking-tight">Login</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Enter your credentials to access your account
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
              name="username"
              render={({ field }) => (
                <FormItem className="gap-2">
                  <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Username
                  </FormLabel>
                  <FormControl>
                    <Input className="h-12" placeholder="Enter username" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem className="gap-2">
                  <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Password
                  </FormLabel>
                  <FormControl>
                    <Input className="h-12" type="password" placeholder="Enter password" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" className="mt-1 h-12 w-full">
              Sign in
            </Button>
          </form>
        </Form>
      </div>
    </div>
  );
}

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { isAxiosError } from 'axios';
import { format } from 'date-fns';
import { toast } from 'sonner';

import { adminApi } from '@/lib/admin-api';
import type { RowError, CreateUserRequest } from '@/lib/admin-api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';

const createUserSchema = z.object({
  id: z.string().min(1, 'ID is required'),
  full_name: z.string().min(1, 'Full name is required'),
  role: z.enum(['student', 'lecturer']),
  dob: z.string().regex(/^(0[1-9]|[12][0-9]|3[01])\/(0[1-9]|1[0-2])\/\d{4}$/, 'Format must be DD/MM/YYYY'),
});

export default function Accounts() {
  const queryClient = useQueryClient();
  const [roleFilter, setRoleFilter] = useState<string>('_all');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(0);
  const pageSize = 50;

  const [createOpen, setCreateOpen] = useState(false);
  const [importErrs, setImportErrs] = useState<RowError[]>([]);

  const { data, isLoading } = useQuery({
    queryKey: ['accounts', roleFilter, search, page],
    queryFn: () => adminApi.listUsers({
      role: roleFilter === '_all' ? undefined : roleFilter,
      search: search || undefined,
      limit: pageSize,
      offset: page * pageSize,
    }),
  });

  const createForm = useForm<z.infer<typeof createUserSchema>>({
    resolver: zodResolver(createUserSchema),
    defaultValues: { role: 'student', id: '', full_name: '', dob: '' },
  });

  const createMutation = useMutation({
    mutationFn: adminApi.createUser,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      toast.success('Account created successfully');
      setCreateOpen(false);
      createForm.reset();
    },
    onError: (err) => {
      if (isAxiosError(err) && err.response?.data?.error?.message) {
        toast.error(err.response.data.error.message);
      } else {
        toast.error('Failed to create account');
      }
    },
  });

  const importMutation = useMutation({
    mutationFn: ({ role, file }: { role: 'student' | 'lecturer', file: File }) => adminApi.importAccounts(role, file),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      toast.success(`Imported ${res.imported} accounts successfully`);
      setImportErrs([]);
    },
    onError: (err) => {
      if (isAxiosError(err) && err.response?.status === 422) {
        setImportErrs(err.response.data.errors);
        toast.error('Import failed: Validation errors found');
      } else {
        toast.error('Failed to import accounts');
      }
    },
  });

  const resetMutation = useMutation({
    mutationFn: adminApi.resetPassword,
    onSuccess: () => {
      toast.success('Password reset to default');
    },
    onError: () => {
      toast.error('Failed to reset password');
    },
  });

  const onImportFile = (role: 'student' | 'lecturer', e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      importMutation.mutate({ role, file });
    }
    e.target.value = '';
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Accounts</h2>
          <p className="text-muted-foreground">Manage student and lecturer accounts.</p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <label className="cursor-pointer">
            <Button render={<span />} variant="outline" disabled={importMutation.isPending}>
              Import Students
            </Button>
            <input type="file" className="hidden" accept=".csv" onChange={(e) => onImportFile('student', e)} />
          </label>
          <label className="cursor-pointer">
            <Button render={<span />} variant="outline" disabled={importMutation.isPending}>
              Import Lecturers
            </Button>
            <input type="file" className="hidden" accept=".csv" onChange={(e) => onImportFile('lecturer', e)} />
          </label>
          
          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger render={<Button />}>
              Create Manual
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Account</DialogTitle>
              </DialogHeader>
              <Form {...createForm}>
                <form onSubmit={createForm.handleSubmit((v) => createMutation.mutate(v as CreateUserRequest))} className="space-y-4">
                  <FormField
                    control={createForm.control}
                    name="role"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Role</FormLabel>
                        <Select onValueChange={field.onChange} defaultValue={field.value}>
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="Select role" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="student">Student</SelectItem>
                            <SelectItem value="lecturer">Lecturer</SelectItem>
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={createForm.control}
                    name="id"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>ID (Username)</FormLabel>
                        <FormControl><Input placeholder="S12345" {...field} /></FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={createForm.control}
                    name="full_name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Full Name</FormLabel>
                        <FormControl><Input placeholder="John Doe" {...field} /></FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={createForm.control}
                    name="dob"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Date of Birth (DD/MM/YYYY)</FormLabel>
                        <FormControl><Input placeholder="01/01/2000" {...field} /></FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <Button type="submit" disabled={createMutation.isPending} className="w-full">
                    {createMutation.isPending ? 'Creating...' : 'Create'}
                  </Button>
                </form>
              </Form>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {importErrs.length > 0 && (
        <div className="border border-destructive rounded-md overflow-hidden">
          <div className="bg-destructive/10 p-3 text-destructive font-medium border-b border-destructive">
            CSV Import Failed - {importErrs.length} validation errors
          </div>
          <div className="max-h-64 overflow-y-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Row</TableHead>
                  <TableHead>Field</TableHead>
                  <TableHead>Message</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {importErrs.map((e, idx) => (
                  <TableRow key={idx}>
                    <TableCell>#{e.row}</TableCell>
                    <TableCell className="font-mono text-xs">{e.field}</TableCell>
                    <TableCell>{e.message}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}

      <div className="flex flex-col sm:flex-row gap-4">
        <Input 
          placeholder="Search ID or name..." 
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-sm"
        />
        <Select value={roleFilter} onValueChange={setRoleFilter}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Filter role" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="_all">All Roles</SelectItem>
            <SelectItem value="student">Student</SelectItem>
            <SelectItem value="lecturer">Lecturer</SelectItem>
            <SelectItem value="admin">Admin</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID / Username</TableHead>
              <TableHead>Full Name</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>DOB</TableHead>
              <TableHead>Joined</TableHead>
              <TableHead className="w-[100px]"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={7} className="text-center">Loading...</TableCell></TableRow>
            ) : data?.data.length === 0 ? (
              <TableRow><TableCell colSpan={7} className="text-center">No accounts found.</TableCell></TableRow>
            ) : (
              data?.data.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="font-medium">{u.username}</TableCell>
                  <TableCell>{u.full_name}</TableCell>
                  <TableCell><Badge variant="secondary">{u.role}</Badge></TableCell>
                  <TableCell>
                    {u.must_change_password ? (
                      <Badge variant="outline" className="text-yellow-600 border-yellow-600">Needs Password Change</Badge>
                    ) : (
                      <Badge variant="outline" className="text-green-600 border-green-600">Active</Badge>
                    )}
                  </TableCell>
                  <TableCell>{u.dob}</TableCell>
                  <TableCell>{format(new Date(u.created_at), 'dd/MM/yyyy')}</TableCell>
                  <TableCell>
                    <AlertDialog>
                      <AlertDialogTrigger render={<Button variant="ghost" size="sm" className="text-destructive" />}>
                        Reset Pwd
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Reset Password?</AlertDialogTitle>
                          <AlertDialogDescription>
                            This will reset the password for {u.username} back to their date of birth (DDMMYYYY). They will be forced to change it on their next login.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction 
                            onClick={() => resetMutation.mutate(u.id)}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                          >
                            Reset Password
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
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
            Showing {page * pageSize + 1} to {Math.min((page + 1) * pageSize, data.total)} of {data.total}
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

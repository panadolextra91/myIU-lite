import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { isAxiosError } from 'axios';
import { format } from 'date-fns';
import { toast } from 'sonner';
import { Search, Upload, Plus } from 'lucide-react';

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
    <div className="flex flex-col gap-6">
      {/* Content Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-3xl font-normal tracking-tight">Accounts</h1>
          <p className="mt-1 italic text-muted-foreground">Manage student and lecturer accounts.</p>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <label className="cursor-pointer">
            <Button render={<span />} variant="outline" disabled={importMutation.isPending}>
              <Upload className="size-4" strokeWidth={1.5} />
              Import Students
            </Button>
            <input type="file" className="hidden" accept=".csv" onChange={(e) => onImportFile('student', e)} />
          </label>
          <label className="cursor-pointer">
            <Button render={<span />} variant="outline" disabled={importMutation.isPending}>
              <Upload className="size-4" strokeWidth={1.5} />
              Import Lecturers
            </Button>
            <input type="file" className="hidden" accept=".csv" onChange={(e) => onImportFile('lecturer', e)} />
          </label>

          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger render={<Button />}>
              <Plus className="size-4" strokeWidth={1.5} />
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
                        <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Role</FormLabel>
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
                        <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">ID (Username)</FormLabel>
                        <FormControl><Input className="font-mono" placeholder="S12345" {...field} /></FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={createForm.control}
                    name="full_name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Full Name</FormLabel>
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
                        <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Date of Birth (DD/MM/YYYY)</FormLabel>
                        <FormControl><Input className="font-mono" placeholder="01/01/2000" {...field} /></FormControl>
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

      {/* CSV import error panel (conditional) */}
      {importErrs.length > 0 && (
        <div className="overflow-hidden rounded-lg border border-destructive">
          <div className="border-b border-destructive bg-destructive/10 p-3 font-medium text-destructive">
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
                    <TableCell className="font-mono tabular-nums">#{e.row}</TableCell>
                    <TableCell className="font-mono text-xs">{e.field}</TableCell>
                    <TableCell>{e.message}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}

      {/* Filter Bar */}
      <div className="flex flex-col gap-4 rounded-lg border bg-card p-3 sm:flex-row">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" strokeWidth={1.5} />
          <Input
            placeholder="Search ID or name..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="border-transparent bg-transparent pl-9 shadow-none"
          />
        </div>
        <Select value={roleFilter} onValueChange={setRoleFilter}>
          <SelectTrigger className="w-full border-transparent bg-transparent shadow-none sm:w-48">
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

      {/* Table Container */}
      <div className="overflow-hidden rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="text-right">ID / Username</TableHead>
              <TableHead>Full Name</TableHead>
              <TableHead>Role</TableHead>
              <TableHead className="text-center">Status</TableHead>
              <TableHead className="text-right">DOB</TableHead>
              <TableHead className="text-right">Joined</TableHead>
              <TableHead className="w-[120px] text-center">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={7} className="text-center text-muted-foreground">Loading...</TableCell></TableRow>
            ) : data?.data.length === 0 ? (
              <TableRow><TableCell colSpan={7} className="text-center text-muted-foreground">No accounts found.</TableCell></TableRow>
            ) : (
              data?.data.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="text-right font-mono font-medium tabular-nums">{u.username}</TableCell>
                  <TableCell className="font-heading text-lg">{u.full_name}</TableCell>
                  <TableCell><Badge variant="outline" className="uppercase">{u.role}</Badge></TableCell>
                  <TableCell className="text-center">
                    {u.must_change_password ? (
                      <Badge variant="warning">Needs Password Change</Badge>
                    ) : (
                      <Badge variant="success">Active</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-right font-mono tabular-nums">{u.dob}</TableCell>
                  <TableCell className="text-right font-mono tabular-nums">{format(new Date(u.created_at), 'dd/MM/yyyy')}</TableCell>
                  <TableCell className="text-center">
                    <AlertDialog>
                      <AlertDialogTrigger render={<Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" />}>
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

        {data && data.total > pageSize && (
          <div className="flex items-center justify-between border-t px-6 py-4">
            <p className="text-sm text-muted-foreground">
              Showing <span className="font-mono tabular-nums">{page * pageSize + 1}</span> to <span className="font-mono tabular-nums">{Math.min((page + 1) * pageSize, data.total)}</span> of <span className="font-mono tabular-nums">{data.total}</span> accounts
            </p>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page === 0} onClick={() => setPage(p => p - 1)}>Previous</Button>
              <Button variant="outline" size="sm" disabled={(page + 1) * pageSize >= data.total} onClick={() => setPage(p => p + 1)}>Next</Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

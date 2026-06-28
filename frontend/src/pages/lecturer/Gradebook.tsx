/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from 'react';
import { useParams } from 'react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Plus, Trash2 } from 'lucide-react';
import { gradesApi } from '@/lib/grades-api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Skeleton } from '@/components/ui/skeleton';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { toast } from 'sonner';

const compSchema = z.object({
  name: z.string().min(1, 'Name required'),
  weight: z.coerce.number().min(0.01).max(100),
  source_type: z.enum(['AUTO', 'MANUAL']).optional(),
  auto_kind: z.enum(['QUIZ_AVERAGE', 'ASSIGNMENT_AVERAGE', '']).optional(),
  parent_index: z.coerce.number().optional(),
});

const schemeSchema = z.object({
  components: z.array(compSchema).min(1),
});

type FormValues = z.infer<typeof schemeSchema>;

export default function LecturerGradebook() {
  const { id } = useParams();
  const courseId = Number(id);
  const [open, setOpen] = useState(false);
  const queryClient = useQueryClient();

  const { data: scheme, isLoading: loadingScheme } = useQuery({
    queryKey: ['scheme', courseId],
    queryFn: () => gradesApi.getScheme(courseId),
    retry: false,
  });

  const { data: grades } = useQuery({
    queryKey: ['grades', courseId],
    queryFn: () => gradesApi.getCourseGrades(courseId),
    enabled: !!scheme,
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(schemeSchema) as any,
    defaultValues: { components: [{ name: '', weight: 100, source_type: 'MANUAL', auto_kind: '' }] },
  });
  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "components",
  });

  const createMutation = useMutation({
    mutationFn: (values: FormValues) => {
      // clean up empty auto_kind
      const cleaned = values.components.map(c => ({
        ...c,
        auto_kind: !c.auto_kind ? undefined : c.auto_kind,
        source_type: !c.source_type ? undefined : c.source_type,
      }));
      return gradesApi.createScheme(courseId, { components: cleaned });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['scheme', courseId] });
      toast.success('Grade scheme created');
      setOpen(false);
    },
    onError: (err: any) => {
      toast.error(err.response?.data?.error?.message || 'Failed to create scheme');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => gradesApi.deleteScheme(courseId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['scheme', courseId] });
      queryClient.invalidateQueries({ queryKey: ['grades', courseId] });
      toast.success('Scheme deleted');
    },
    onError: (err: any) => {
      toast.error(err.response?.data?.error?.message || 'Failed to delete scheme');
    },
  });

  const uploadMutation = useMutation({
    mutationFn: ({ componentId, file }: { componentId: number; file: File }) =>
      gradesApi.importScoresCSV(courseId, componentId, file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['grades', courseId] });
      toast.success('Scores imported successfully');
    },
    onError: (err: any) => {
      const errs = err.response?.data?.errors;
      if (errs && Array.isArray(errs)) {
        toast.error(`Import failed: ${errs.length} row errors. Example: Row ${errs[0].row} - ${errs[0].message}`);
      } else {
        toast.error(err.response?.data?.error?.message || 'Failed to import scores');
      }
    },
  });

  const publishMutation = useMutation({
    mutationFn: (componentId: number) => gradesApi.publishComponent(courseId, componentId),
    onSuccess: () => {
      toast.success('Grades published and notifications sent!');
    },
    onError: (err: any) => {
      toast.error(err.response?.data?.error?.message || 'Failed to publish grades');
    },
  });

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>, compId: number) => {
    if (e.target.files && e.target.files.length > 0) {
      uploadMutation.mutate({ componentId: compId, file: e.target.files[0] });
      e.target.value = ''; // reset input
    }
  };

  if (loadingScheme) {
    return <div className="p-8"><Skeleton className="h-[400px] w-full" /></div>;
  }

  return (
    <div className="p-8 flex flex-col gap-12">
      {/* Title Row */}
      <div className="flex items-end justify-between">
        <h1 className="font-heading text-4xl font-normal tracking-tight text-foreground">Gradebook</h1>
        <div className="flex items-center gap-4">
          {scheme && (
            <Button
              variant="outline"
              onClick={() => deleteMutation.mutate()}
              className="h-11 border-destructive px-5 text-xs font-medium uppercase tracking-widest text-destructive hover:bg-destructive/10 hover:text-destructive"
            >
              <Trash2 strokeWidth={1.5} />
              Delete Scheme
            </Button>
          )}
          {!scheme && (
            <Dialog open={open} onOpenChange={setOpen}>
              <DialogTrigger
                render={
                  <Button className="h-11 px-5 text-xs font-medium uppercase tracking-widest" />
                }
              >
                <Plus strokeWidth={1.5} />
                Create Scheme
              </DialogTrigger>
              <DialogContent className="max-h-[80vh] max-w-2xl overflow-y-auto">
                <DialogHeader>
                  <DialogTitle className="font-heading text-2xl font-normal tracking-tight">
                    Create Grade Scheme
                  </DialogTitle>
                </DialogHeader>
                <Form {...form}>
                  <form onSubmit={form.handleSubmit((v) => createMutation.mutate(v))} className="space-y-4">
                    {fields.map((field, index) => (
                      <div key={field.id} className="flex items-end gap-2 rounded-lg border bg-muted/20 p-3">
                        <FormField
                          control={form.control}
                          name={`components.${index}.name`}
                          render={({ field }) => (
                            <FormItem className="flex-1">
                              <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Name</FormLabel>
                              <FormControl><Input {...field} /></FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name={`components.${index}.weight`}
                          render={({ field }) => (
                            <FormItem className="w-20">
                              <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Weight</FormLabel>
                              <FormControl><Input type="number" step="0.1" className="font-mono tabular-nums" {...field} /></FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name={`components.${index}.source_type`}
                          render={({ field }) => (
                            <FormItem className="w-28">
                              <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Source</FormLabel>
                              <Select onValueChange={field.onChange} defaultValue={field.value}>
                                <FormControl>
                                  <SelectTrigger><SelectValue placeholder="Select" /></SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                  <SelectItem value="MANUAL">MANUAL</SelectItem>
                                  <SelectItem value="AUTO">AUTO</SelectItem>
                                  <SelectItem value="composite">(Composite)</SelectItem>
                                </SelectContent>
                              </Select>
                            </FormItem>
                          )}
                        />
                        {form.watch(`components.${index}.source_type`) === 'AUTO' && (
                          <FormField
                            control={form.control}
                            name={`components.${index}.auto_kind`}
                            render={({ field }) => (
                              <FormItem className="w-40">
                                <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Auto Kind</FormLabel>
                                <Select onValueChange={field.onChange} defaultValue={field.value}>
                                  <FormControl>
                                    <SelectTrigger><SelectValue placeholder="Select" /></SelectTrigger>
                                  </FormControl>
                                  <SelectContent>
                                    <SelectItem value="QUIZ_AVERAGE">Quizzes</SelectItem>
                                    <SelectItem value="ASSIGNMENT_AVERAGE">Assignments</SelectItem>
                                  </SelectContent>
                                </Select>
                              </FormItem>
                            )}
                          />
                        )}
                        <FormField
                          control={form.control}
                          name={`components.${index}.parent_index`}
                          render={({ field }) => (
                            <FormItem className="w-24">
                              <FormLabel className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Parent Idx</FormLabel>
                              <FormControl><Input type="number" className="font-mono tabular-nums" {...field} value={field.value ?? ''} /></FormControl>
                            </FormItem>
                          )}
                        />
                        <Button type="button" variant="ghost" size="icon" onClick={() => remove(index)}>
                          <Trash2 strokeWidth={1.5} />
                        </Button>
                      </div>
                    ))}
                    <Button type="button" variant="secondary" onClick={() => append({ name: '', weight: 0, source_type: 'MANUAL', auto_kind: '' })}>
                      <Plus strokeWidth={1.5} />
                      Add Component
                    </Button>
                    <div className="flex justify-end pt-4">
                      <Button type="submit" disabled={createMutation.isPending}>Save Scheme</Button>
                    </div>
                  </form>
                </Form>
              </DialogContent>
            </Dialog>
          )}
        </div>
      </div>

      {scheme && (
        <div className="flex flex-col gap-12">
          {/* Section 1: Scheme Structure */}
          <section className="rounded-lg border bg-card p-8">
            <h2 className="mb-6 font-heading text-2xl font-normal tracking-tight text-foreground">Scheme Structure</h2>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Component</TableHead>
                    <TableHead className="text-right">Weight</TableHead>
                    <TableHead>Source</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {scheme.components.map(c => (
                    <TableRow key={c.id} className={c.parent_id ? 'bg-muted/20' : undefined}>
                      <TableCell className={c.parent_id ? 'pl-8 italic text-muted-foreground' : 'font-medium text-foreground'}>
                        {c.parent_id ? `— ${c.name}` : c.name}
                      </TableCell>
                      <TableCell className={`text-right font-mono tabular-nums ${c.parent_id ? 'text-muted-foreground' : ''}`}>{c.weight}%</TableCell>
                      <TableCell>
                        {c.parent_id ? (
                          <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70">
                            {c.source_type || 'Composite'} {c.auto_kind ? `(${c.auto_kind})` : ''}
                          </span>
                        ) : (
                          <Badge variant="secondary" className="text-[11px] font-bold tracking-wider">
                            {c.source_type || 'Composite'} {c.auto_kind ? `(${c.auto_kind})` : ''}
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-3">
                          {c.source_type === 'MANUAL' && (
                            <Input
                              type="file"
                              accept=".csv"
                              onChange={(e) => handleFileUpload(e, c.id)}
                              disabled={uploadMutation.isPending}
                              className="w-44"
                            />
                          )}
                          {!c.parent_id && (
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={publishMutation.isPending}
                              onClick={() => publishMutation.mutate(c.id)}
                              className="border-primary text-xs uppercase tracking-wider text-primary hover:bg-primary hover:text-primary-foreground"
                            >
                              Publish
                            </Button>
                          )}
                          {c.source_type !== 'MANUAL' && c.parent_id && (
                            <span className="italic text-muted-foreground">—</span>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </section>

          {/* Section 2: Live Overall Grades */}
          {grades && grades.length > 0 && (
            <section className="rounded-lg border bg-card p-8">
              <h2 className="mb-6 font-heading text-2xl font-normal tracking-tight text-foreground">Live Overall Grades</h2>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Student ID</TableHead>
                      {scheme.components.filter(c => !c.parent_id).map(c => (
                        <TableHead key={c.id} className="text-right">{c.name}</TableHead>
                      ))}
                      <TableHead className="text-right text-primary">Overall</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {grades.map(g => (
                      <TableRow key={g.student_id}>
                        <TableCell className="font-mono tabular-nums text-muted-foreground">{g.student_id}</TableCell>
                        {scheme.components.filter(c => !c.parent_id).map(c => {
                          const score = g.components.find(gc => gc.component_id === c.id)?.score || 0;
                          return <TableCell key={c.id} className="text-right font-mono tabular-nums">{score}</TableCell>;
                        })}
                        <TableCell className="text-right font-mono tabular-nums font-bold text-primary">{g.overall}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </section>
          )}
        </div>
      )}
    </div>
  );
}

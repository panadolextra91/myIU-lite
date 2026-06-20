/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from 'react';
import { useParams } from 'react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { gradesApi } from '@/lib/grades-api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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
    <div className="p-8 space-y-8">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold">Gradebook</h1>
        <div className="flex gap-4">
          {scheme && (
            <Button variant="destructive" onClick={() => deleteMutation.mutate()}>
              Delete Scheme
            </Button>
          )}
          {!scheme && (
            <Dialog open={open} onOpenChange={setOpen}>
              <DialogTrigger render={<Button />}>
                Create Scheme
              </DialogTrigger>
              <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
                <DialogHeader>
                  <DialogTitle>Create Grade Scheme</DialogTitle>
                </DialogHeader>
                <Form {...form}>
                  <form onSubmit={form.handleSubmit((v) => createMutation.mutate(v))} className="space-y-4">
                    {fields.map((field, index) => (
                      <div key={field.id} className="flex gap-2 items-end border p-2 rounded">
                        <FormField
                          control={form.control}
                          name={`components.${index}.name`}
                          render={({ field }) => (
                            <FormItem className="flex-1">
                              <FormLabel>Name</FormLabel>
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
                              <FormLabel>Weight</FormLabel>
                              <FormControl><Input type="number" step="0.1" {...field} /></FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name={`components.${index}.source_type`}
                          render={({ field }) => (
                            <FormItem className="w-28">
                              <FormLabel>Source</FormLabel>
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
                                <FormLabel>Auto Kind</FormLabel>
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
                              <FormLabel>Parent Idx</FormLabel>
                              <FormControl><Input type="number" {...field} value={field.value ?? ''} /></FormControl>
                            </FormItem>
                          )}
                        />
                        <Button type="button" variant="ghost" onClick={() => remove(index)}>X</Button>
                      </div>
                    ))}
                    <Button type="button" variant="secondary" onClick={() => append({ name: '', weight: 0, source_type: 'MANUAL', auto_kind: '' })}>
                      Add Component
                    </Button>
                    <div className="pt-4 flex justify-end">
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
        <div className="space-y-6">
          <div className="border rounded-lg p-4">
            <h2 className="text-xl font-semibold mb-4">Scheme Structure</h2>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Component</TableHead>
                  <TableHead>Weight</TableHead>
                  <TableHead>Source</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {scheme.components.map(c => (
                  <TableRow key={c.id}>
                    <TableCell>{c.parent_id ? `— ${c.name}` : c.name}</TableCell>
                    <TableCell>{c.weight}%</TableCell>
                    <TableCell>{c.source_type || 'Composite'} {c.auto_kind ? `(${c.auto_kind})` : ''}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {c.source_type === 'MANUAL' && (
                          <Input
                          type="file"
                          accept=".csv"
                          onChange={(e) => handleFileUpload(e, c.id)}
                          disabled={uploadMutation.isPending}
                        />
                        )}
                        {!c.parent_id && (
                          <Button 
                            variant="secondary" 
                            size="sm"
                            disabled={publishMutation.isPending}
                            onClick={() => publishMutation.mutate(c.id)}
                          >
                            Publish
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {grades && grades.length > 0 && (
            <div className="border rounded-lg p-4">
              <h2 className="text-xl font-semibold mb-4">Live Overall Grades</h2>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Student ID</TableHead>
                    {scheme.components.filter(c => !c.parent_id).map(c => (
                      <TableHead key={c.id}>{c.name}</TableHead>
                    ))}
                    <TableHead className="font-bold">Overall</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {grades.map(g => (
                    <TableRow key={g.student_id}>
                      <TableCell>{g.student_id}</TableCell>
                      {scheme.components.filter(c => !c.parent_id).map(c => {
                        const score = g.components.find(gc => gc.component_id === c.id)?.score || 0;
                        return <TableCell key={c.id}>{score}</TableCell>;
                      })}
                      <TableCell className="font-bold text-lg">{g.overall}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

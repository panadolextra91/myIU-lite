/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useParams } from 'react-router';
import { Plus, Upload, Trash2, Calendar, ArrowRight, Save } from 'lucide-react';
import { courseworkApi, type UIQuestionRequest, type UIOptionRequest } from '@/lib/coursework-api';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Checkbox } from '@/components/ui/checkbox';
import { toast } from 'sonner';

const configSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  pool_size: z.number().min(1),
  max_questions: z.number().min(1),
  max_grade: z.number().min(0),
  shuffle: z.boolean(),
  retake_count: z.number().min(1),
  open_at: z.string().optional(),
  close_at: z.string().optional(),
}).refine(data => data.max_questions <= data.pool_size, {
  message: 'Max questions cannot exceed pool size',
  path: ['max_questions'],
});

export default function LecturerQuizzes() {
  const { id } = useParams<{ id: string }>();
  const courseId = Number(id);
  const queryClient = useQueryClient();

  const [createOpen, setCreateOpen] = useState(false);
  const [activeQuizId, setActiveQuizId] = useState<number | null>(null);
  const [csvFile, setCsvFile] = useState<File | null>(null);

  const [uiPrompt, setUiPrompt] = useState('');
  const [uiType, setUiType] = useState<'single' | 'multi'>('single');
  const [uiOptions, setUiOptions] = useState<UIOptionRequest[]>([
    { text: '', is_correct: false },
    { text: '', is_correct: false },
  ]);

  const { data: quizzes, isLoading } = useQuery({
    queryKey: ['quizzes', courseId],
    queryFn: () => courseworkApi.listQuizzes(courseId),
  });

  const configForm = useForm<z.infer<typeof configSchema>>({
    resolver: zodResolver(configSchema),
    defaultValues: {
      title: '',
      pool_size: 10,
      max_questions: 10,
      max_grade: 100,
      shuffle: true,
      retake_count: 1,
      open_at: '',
      close_at: '',
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: z.infer<typeof configSchema>) => courseworkApi.createQuiz(courseId, {
      ...data,
      open_at: data.open_at || undefined,
      close_at: data.close_at || undefined,
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quizzes', courseId] });
      setCreateOpen(false);
      configForm.reset();
      toast.success('Quiz created successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error?.message || 'Failed to create quiz');
    },
  });

  const importMutation = useMutation({
    mutationFn: ({ quizId, file }: { quizId: number; file: File }) => courseworkApi.importQuizCSV(courseId, quizId, file),
    onSuccess: () => {
      setCsvFile(null);
      toast.success('Questions imported successfully');
    },
    onError: (error: any) => {
      toast.error('Failed to import CSV. Check console for row errors.');
      console.error(error.response?.data?.errors);
    },
  });

  const addQuestionMutation = useMutation({
    mutationFn: (data: UIQuestionRequest) => courseworkApi.addUIQuestion(courseId, activeQuizId!, data),
    onSuccess: () => {
      toast.success('Question added successfully');
      setUiPrompt('');
      setUiOptions([{ text: '', is_correct: false }, { text: '', is_correct: false }]);
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error?.message || 'Failed to add question');
    },
  });

  const onSubmitConfig = (values: z.infer<typeof configSchema>) => {
    createMutation.mutate(values);
  };

  const handleAddOption = () => {
    setUiOptions([...uiOptions, { text: '', is_correct: false }]);
  };

  const handleRemoveOption = (index: number) => {
    setUiOptions(uiOptions.filter((_, i) => i !== index));
  };

  const handleOptionTextChange = (index: number, text: string) => {
    const newOptions = [...uiOptions];
    newOptions[index].text = text;
    setUiOptions(newOptions);
  };

  const handleOptionCorrectChange = (index: number, isCorrect: boolean) => {
    const newOptions = [...uiOptions];
    if (uiType === 'single') {
      newOptions.forEach((o, i) => (o.is_correct = i === index));
    } else {
      newOptions[index].is_correct = isCorrect;
    }
    setUiOptions(newOptions);
  };

  const handleSubmitQuestion = () => {
    if (!activeQuizId) return;
    if (!uiPrompt) {
      toast.error('Prompt cannot be empty');
      return;
    }
    const correctCount = uiOptions.filter((o) => o.is_correct).length;
    if (uiType === 'single' && correctCount !== 1) {
      toast.error('Single choice must have exactly 1 correct option');
      return;
    }
    if (uiType === 'multi' && correctCount < 1) {
      toast.error('Multi choice must have at least 1 correct option');
      return;
    }
    addQuestionMutation.mutate({ prompt: uiPrompt, question_type: uiType, options: uiOptions });
  };

  if (isLoading) {
    return <div className="p-8"><div className="h-32 bg-muted rounded-lg animate-pulse" /></div>;
  }

  return (
    <div className="mx-auto max-w-6xl space-y-12 p-8">
      {/* Header */}
      <section className="flex flex-col gap-4 border-b pb-6 md:flex-row md:items-end md:justify-between">
        <h1 className="text-4xl font-normal tracking-tight text-primary">Quizzes</h1>
        <Button onClick={() => setCreateOpen(true)} className="h-11">
          <Plus className="mr-2 h-4 w-4" strokeWidth={1.5} /> Create Quiz
        </Button>
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogContent className="sm:max-w-[425px]">
            <DialogHeader>
              <DialogTitle>Create New Quiz</DialogTitle>
            </DialogHeader>
            <Form {...configForm}>
              <form onSubmit={configForm.handleSubmit(onSubmitConfig)} className="space-y-4">
                <FormField control={configForm.control} name="title" render={({ field }) => (
                  <FormItem><FormLabel>Title</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
                )} />
                <div className="grid grid-cols-2 gap-4">
                  <FormField control={configForm.control} name="pool_size" render={({ field }) => (
                    <FormItem><FormLabel>Pool Size</FormLabel><FormControl><Input type="number" {...field} onChange={e => field.onChange(parseInt(e.target.value))} /></FormControl><FormMessage /></FormItem>
                  )} />
                  <FormField control={configForm.control} name="max_questions" render={({ field }) => (
                    <FormItem><FormLabel>Max Questions</FormLabel><FormControl><Input type="number" {...field} onChange={e => field.onChange(parseInt(e.target.value))} /></FormControl><FormMessage /></FormItem>
                  )} />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <FormField control={configForm.control} name="max_grade" render={({ field }) => (
                    <FormItem><FormLabel>Max Grade</FormLabel><FormControl><Input type="number" step="0.1" {...field} onChange={e => field.onChange(parseFloat(e.target.value))} /></FormControl><FormMessage /></FormItem>
                  )} />
                  <FormField control={configForm.control} name="retake_count" render={({ field }) => (
                    <FormItem><FormLabel>Retake Count</FormLabel><FormControl><Input type="number" {...field} onChange={e => field.onChange(parseInt(e.target.value))} /></FormControl><FormMessage /></FormItem>
                  )} />
                </div>
                <FormField control={configForm.control} name="shuffle" render={({ field }) => (
                  <FormItem className="flex items-center justify-between rounded-lg border p-4"><FormLabel>Shuffle Questions</FormLabel><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl></FormItem>
                )} />
                <div className="grid grid-cols-2 gap-4">
                  <FormField control={configForm.control} name="open_at" render={({ field }) => (
                    <FormItem><FormLabel>Open At</FormLabel><FormControl><Input type="datetime-local" {...field} /></FormControl><FormMessage /></FormItem>
                  )} />
                  <FormField control={configForm.control} name="close_at" render={({ field }) => (
                    <FormItem><FormLabel>Close At</FormLabel><FormControl><Input type="datetime-local" {...field} /></FormControl><FormMessage /></FormItem>
                  )} />
                </div>
                <Button type="submit" className="w-full" disabled={createMutation.isPending}>Create</Button>
              </form>
            </Form>
          </DialogContent>
        </Dialog>
      </section>

      {/* Quiz grid */}
      <section className="grid grid-cols-1 gap-6 md:grid-cols-2">
        {quizzes?.map((quiz) => (
          <Card
            key={quiz.id}
            onClick={() => setActiveQuizId(quiz.id)}
            className="group relative cursor-pointer overflow-hidden transition-colors hover:border-primary"
          >
            <div className="pointer-events-none absolute right-4 top-4 opacity-0 transition-opacity group-hover:opacity-100">
              <ArrowRight className="h-5 w-5 text-primary" strokeWidth={1.5} />
            </div>
            <CardHeader>
              <CardTitle className="text-2xl font-normal tracking-tight">{quiz.title}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap gap-x-6 gap-y-2 font-mono text-sm tabular-nums text-muted-foreground">
                <span>Pool: {quiz.pool_size}</span>
                <span>Questions: {quiz.max_questions}</span>
                <span>Max Grade: {quiz.max_grade}</span>
              </div>
              <p className="flex items-center gap-2 text-sm text-primary">
                <Calendar className="h-4 w-4" strokeWidth={1.5} />
                <span className="font-mono tabular-nums">
                  Available: {quiz.open_at ? new Date(quiz.open_at).toLocaleString() : 'No open date'} - {quiz.close_at ? new Date(quiz.close_at).toLocaleString() : 'No close date'}
                </span>
              </p>
            </CardContent>
          </Card>
        ))}
        {quizzes?.length === 0 && (
          <div className="col-span-full py-12 text-center text-muted-foreground">No quizzes created yet</div>
        )}
      </section>

      {/* Authoring panel */}
      {activeQuizId && (
        <Card className="border-t-4 border-t-primary">
          <CardHeader>
            <CardTitle className="text-3xl font-normal tracking-tight">
              Author Questions for Quiz <span className="font-mono tabular-nums">#{activeQuizId}</span>
            </CardTitle>
            <div className="mt-4 h-px w-16 bg-border" />
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 gap-12 lg:grid-cols-12">
              {/* (a) Import via CSV */}
              <div className="lg:col-span-4">
                <h3 className="mb-4 text-xs font-medium uppercase tracking-wider text-muted-foreground">Import via CSV</h3>
                <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-border bg-muted/40 p-6 text-center">
                  <Label
                    htmlFor="csv-upload"
                    className="cursor-pointer text-xs font-medium uppercase tracking-wider text-muted-foreground transition-colors hover:text-primary"
                  >
                    {csvFile ? csvFile.name : 'Select File'}
                  </Label>
                  <input
                    id="csv-upload"
                    type="file"
                    accept=".csv"
                    className="hidden"
                    onChange={(e) => setCsvFile(e.target.files?.[0] || null)}
                  />
                  <Upload className="h-8 w-8 text-muted-foreground" strokeWidth={1.5} />
                  <p className="max-w-[220px] text-sm italic text-muted-foreground">
                    Format: question,A,B,C,D,correct (where correct is A, B, C, or D)
                  </p>
                  <Button
                    size="sm"
                    className="mt-2"
                    onClick={() => csvFile && importMutation.mutate({ quizId: activeQuizId, file: csvFile })}
                    disabled={!csvFile || importMutation.isPending}
                  >
                    <Upload className="mr-2 h-4 w-4" strokeWidth={1.5} /> Import CSV
                  </Button>
                </div>
              </div>

              {/* (b) Add UI Question */}
              <div className="space-y-6 border-t pt-6 lg:col-span-8 lg:border-l lg:border-t-0 lg:pl-12 lg:pt-0">
                <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Add UI Question</h3>

                <div className="space-y-2">
                  <Label htmlFor="ui-prompt" className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Question Text</Label>
                  <Input id="ui-prompt" placeholder="Question prompt..." value={uiPrompt} onChange={(e) => setUiPrompt(e.target.value)} />
                </div>

                <RadioGroup value={uiType} onValueChange={(v) => setUiType(v as 'single' | 'multi')} className="flex gap-6">
                  <div className="flex items-center gap-2">
                    <RadioGroupItem value="single" id="single" />
                    <Label htmlFor="single" className="cursor-pointer font-normal">Single Choice</Label>
                  </div>
                  <div className="flex items-center gap-2">
                    <RadioGroupItem value="multi" id="multi" />
                    <Label htmlFor="multi" className="cursor-pointer font-normal">Multi Choice</Label>
                  </div>
                </RadioGroup>

                <div className="space-y-3">
                  <Label className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Options</Label>
                  {uiOptions.map((opt, i) => (
                    <div key={i} className="flex items-center gap-3">
                      {uiType === 'single' ? (
                        <RadioGroup value={opt.is_correct ? String(i) : ''} onValueChange={() => handleOptionCorrectChange(i, true)}>
                          <RadioGroupItem value={String(i)} id={`opt-${i}`} />
                        </RadioGroup>
                      ) : (
                        <Checkbox checked={opt.is_correct} onCheckedChange={(c) => handleOptionCorrectChange(i, !!c)} />
                      )}
                      <Input className="flex-grow" value={opt.text} onChange={(e) => handleOptionTextChange(i, e.target.value)} placeholder={`Option ${i + 1}`} />
                      {uiOptions.length > 2 && (
                        <Button variant="ghost" size="icon" onClick={() => handleRemoveOption(i)}>
                          <Trash2 className="h-4 w-4 text-muted-foreground transition-colors hover:text-destructive" strokeWidth={1.5} />
                        </Button>
                      )}
                    </div>
                  ))}
                  <Button variant="outline" size="sm" onClick={handleAddOption}>
                    <Plus className="mr-2 h-4 w-4" strokeWidth={1.5} /> Add Option
                  </Button>
                </div>

                <Button onClick={handleSubmitQuestion} disabled={addQuestionMutation.isPending} className="h-12 w-full">
                  <Save className="mr-2 h-4 w-4" strokeWidth={1.5} /> Save Question
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

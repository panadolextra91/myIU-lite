/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { courseworkApi, type StudentQuizAttemptView } from '@/lib/coursework-api';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Checkbox } from '@/components/ui/checkbox';
import { Skeleton } from '@/components/ui/skeleton';
import { ArrowLeft, ListChecks, Award, Inbox } from 'lucide-react';
import { toast } from 'sonner';

export default function StudentQuizzes() {
  const { id } = useParams<{ id: string }>();
  const courseId = Number(id);

  const [activeQuizId, setActiveQuizId] = useState<number | null>(null);
  const [activeAttempt, setActiveAttempt] = useState<StudentQuizAttemptView | null>(null);
  const [answers, setAnswers] = useState<Record<number, number[]>>({});

  const { data: quizzes, isLoading } = useQuery({
    queryKey: ['studentQuizzes', courseId],
    queryFn: () => courseworkApi.listStudentQuizzes(courseId),
  });

  const startMutation = useMutation({
    mutationFn: (quizId: number) => courseworkApi.startAttempt(courseId, quizId),
    onSuccess: (data, quizId) => {
      setActiveQuizId(quizId);
      setActiveAttempt(data);
      setAnswers(data.selected_options || {});
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error?.message || 'Failed to start attempt');
    },
  });

  const submitMutation = useMutation({
    mutationFn: () => courseworkApi.submitAttempt(courseId, activeQuizId!, activeAttempt!.id, answers),
    onSuccess: (data) => {
      toast.success(`Attempt submitted! Score: ${data.score.toFixed(2)}`);
      // Reload attempt to get the latest state and potentially correct answers
      courseworkApi.getAttempt(courseId, activeQuizId!, activeAttempt!.id).then(setActiveAttempt);
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error?.message || 'Failed to submit attempt');
    },
  });

  const handleToggleOption = (qId: number, optId: number, isSingle: boolean) => {
    if (activeAttempt?.status !== 'IN_PROGRESS') return;
    setAnswers(prev => {
      if (isSingle) {
        return { ...prev, [qId]: [optId] };
      }
      const current = prev[qId] || [];
      if (current.includes(optId)) {
        return { ...prev, [qId]: current.filter(id => id !== optId) };
      }
      return { ...prev, [qId]: [...current, optId] };
    });
  };

  if (isLoading) {
    return (
      <div className="p-8 max-w-6xl mx-auto space-y-8">
        <Skeleton className="h-9 w-64" />
        <div className="grid gap-6 md:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-56 w-full rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  if (activeAttempt && activeQuizId) {
    const isTerminal = activeAttempt.status !== 'IN_PROGRESS';
    const activeQuiz = quizzes?.find((q) => q.id === activeQuizId);
    return (
      <div className="p-8 max-w-5xl mx-auto space-y-12">
        {/* Header Row */}
        <header className="flex justify-between items-end gap-4">
          <div>
            <h1 className="text-3xl tracking-tight text-foreground leading-none">
              Attempt #<span className="font-mono tabular-nums">{activeAttempt.attempt_number}</span>
            </h1>
            {activeQuiz && (
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground mt-3">
                {activeQuiz.title}
              </p>
            )}
          </div>
          <Button
            variant="outline"
            onClick={() => { setActiveAttempt(null); setActiveQuizId(null); }}
          >
            <ArrowLeft strokeWidth={1.5} />
            Back to Quizzes
          </Button>
        </header>

        {/* Result Banner */}
        {isTerminal && (
          <Card className="bg-primary text-primary-foreground border-primary p-6 flex flex-col md:flex-row md:items-center md:justify-between gap-6">
            <div className="flex items-center gap-6">
              <div className="border-r border-primary-foreground/20 pr-6">
                <p className="text-xs font-medium uppercase tracking-wider opacity-80 mb-1">Score</p>
                <span className="font-mono tabular-nums text-5xl leading-none">
                  {activeAttempt.score?.toFixed(2)}
                </span>
              </div>
              <div>
                <h2 className="text-2xl text-primary-foreground">Attempt Result</h2>
                <p className="text-sm opacity-90 max-w-md mt-1">
                  {activeAttempt.correct_options
                    ? 'Window is closed. Correct answers are revealed.'
                    : 'Window is still open. Correct answers will be revealed after the quiz closes.'}
                </p>
              </div>
            </div>
          </Card>
        )}

        {/* Question Stack */}
        <div className="space-y-8">
          {activeAttempt.questions.map((q, idx) => {
            const selected = answers[q.id] || [];
            const corrects = activeAttempt.correct_options?.[q.id];

            return (
              <Card key={q.id} className="p-6 gap-5">
                <div className="flex justify-between items-start gap-4">
                  <h3 className="text-xl text-primary">Question {idx + 1}</h3>
                  {q.question_type === 'multi' && (
                    <Badge variant="secondary">Select all that apply</Badge>
                  )}
                </div>
                <p className="text-base text-foreground leading-relaxed">{q.prompt}</p>
                <div className={q.question_type === 'multi' ? 'grid grid-cols-1 md:grid-cols-2 gap-3' : 'space-y-3'}>
                  {q.options.map((opt) => {
                    const isSelected = selected.includes(opt.id);
                    let borderClass = 'border-border';
                    let bgClass = 'bg-muted/40';

                    if (isTerminal) {
                      if (corrects) {
                        // Window closed: show green for correct options, red for wrong selected
                        const isActuallyCorrect = corrects.includes(opt.id);
                        if (isActuallyCorrect) {
                          bgClass = 'bg-success/10';
                          borderClass = 'border-success';
                        } else if (isSelected) {
                          bgClass = 'bg-destructive/10';
                          borderClass = 'border-destructive';
                        }
                      } else {
                        // Window open: highlight their selection
                        if (isSelected) {
                          bgClass = 'bg-primary/10';
                          borderClass = 'border-primary';
                        }
                      }
                    } else {
                      // In progress: highlight selection softly
                      if (isSelected) {
                        bgClass = 'bg-primary/10';
                        borderClass = 'border-primary';
                      }
                    }

                    return (
                      <div key={opt.id} className={`flex items-center gap-4 p-4 rounded-lg border ${borderClass} ${bgClass} transition-colors`}>
                        {q.question_type === 'single' ? (
                          <RadioGroup value={selected[0]?.toString() || ''} onValueChange={() => handleToggleOption(q.id, opt.id, true)} disabled={isTerminal}>
                            <RadioGroupItem value={opt.id.toString()} id={`opt-${opt.id}`} />
                          </RadioGroup>
                        ) : (
                          <Checkbox checked={isSelected} onCheckedChange={() => handleToggleOption(q.id, opt.id, false)} disabled={isTerminal} />
                        )}
                        <label htmlFor={`opt-${opt.id}`} className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 flex-grow cursor-pointer">
                          {opt.text}
                        </label>
                      </div>
                    );
                  })}
                </div>
              </Card>
            );
          })}
        </div>

        {/* Footer Action */}
        {!isTerminal && (
          <footer className="flex justify-end pt-6 border-t border-border">
            <Button onClick={() => submitMutation.mutate()} disabled={submitMutation.isPending} size="lg">
              Submit Attempt
            </Button>
          </footer>
        )}
      </div>
    );
  }

  return (
    <div className="p-8 max-w-6xl mx-auto space-y-10">
      <h1 className="text-3xl tracking-tight text-foreground">Course Quizzes</h1>

      <div className="grid gap-6 md:grid-cols-2">
        {quizzes?.map((quiz) => {
          const now = new Date();
          const openAt = quiz.open_at ? new Date(quiz.open_at) : null;
          const closeAt = quiz.close_at ? new Date(quiz.close_at) : null;

          let statusText = 'Available';
          let canTake = true;

          if (openAt && now < openAt) {
            statusText = `Opens at ${openAt.toLocaleString()}`;
            canTake = false;
          } else if (closeAt && now > closeAt) {
            statusText = `Closed at ${closeAt.toLocaleString()}`;
            // Let them fetch attempts anyway, but "startAttempt" handles auto-submit and review
          }

          const isUpcoming = !!(openAt && now < openAt);
          const isClosed = !!(closeAt && now > closeAt);
          const statusLabel = isUpcoming ? 'Upcoming' : isClosed ? 'Closed' : 'Available';
          const statusVariant = isUpcoming ? 'secondary' : isClosed ? 'outline' : 'success';

          return (
            <Card key={quiz.id} className="flex flex-col justify-between gap-5 p-6">
              <div className="space-y-3">
                <div className="flex items-start justify-between gap-3">
                  <Badge variant={statusVariant}>{statusLabel}</Badge>
                </div>
                <h2 className="text-2xl text-foreground">{quiz.title}</h2>
                {(isUpcoming || isClosed) && (
                  <p className={`text-sm italic font-mono ${isUpcoming ? 'text-destructive' : 'text-muted-foreground'}`}>
                    {statusText}
                  </p>
                )}
                <div className="flex flex-wrap gap-6 pt-1 text-sm text-muted-foreground">
                  <div className="flex items-center gap-1.5">
                    <ListChecks className="size-4 shrink-0" strokeWidth={1.5} />
                    <span>Questions: <span className="font-mono tabular-nums text-foreground">{quiz.max_questions}</span></span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <Award className="size-4 shrink-0" strokeWidth={1.5} />
                    <span>Max Grade: <span className="font-mono tabular-nums text-foreground">{quiz.max_grade}</span></span>
                  </div>
                </div>
              </div>
              <Button
                onClick={() => startMutation.mutate(quiz.id)}
                disabled={!canTake && !closeAt} // Allow review if closed
                variant={closeAt && now > closeAt ? 'outline' : 'default'}
                className="w-full"
              >
                {closeAt && now > closeAt ? 'Review Attempt' : 'Take / Resume Quiz'}
              </Button>
            </Card>
          );
        })}
        {quizzes?.length === 0 && (
          <Card className="col-span-2 flex flex-col items-center justify-center gap-3 py-16 text-muted-foreground">
            <Inbox className="size-8" strokeWidth={1.5} />
            <p className="text-sm">No quizzes available</p>
          </Card>
        )}
      </div>
    </div>
  );
}

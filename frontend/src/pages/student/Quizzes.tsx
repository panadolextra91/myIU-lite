/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { courseworkApi, type StudentQuizAttemptView } from '@/lib/coursework-api';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Checkbox } from '@/components/ui/checkbox';
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
    return <div className="p-8"><div className="h-32 bg-gray-100 rounded-lg animate-pulse" /></div>;
  }

  if (activeAttempt && activeQuizId) {
    const isTerminal = activeAttempt.status !== 'IN_PROGRESS';
    return (
      <div className="p-8 max-w-4xl mx-auto space-y-8">
        <div className="flex justify-between items-center">
          <h1 className="text-3xl font-bold tracking-tight">Attempt #{activeAttempt.attempt_number}</h1>
          <Button variant="outline" onClick={() => { setActiveAttempt(null); setActiveQuizId(null); }}>
            Back to Quizzes
          </Button>
        </div>

        {isTerminal && (
          <Card className="bg-blue-50 border-blue-200">
            <CardContent className="pt-6">
              <div className="text-xl font-semibold">
                Score: {activeAttempt.score?.toFixed(2)}
              </div>
              <div className="text-sm text-gray-500 mt-1">
                {activeAttempt.correct_options ? 'Window is closed. Correct answers are revealed.' : 'Window is still open. Correct answers will be revealed after the quiz closes.'}
              </div>
            </CardContent>
          </Card>
        )}

        <div className="space-y-6">
          {activeAttempt.questions.map((q, idx) => {
            const selected = answers[q.id] || [];
            const corrects = activeAttempt.correct_options?.[q.id];

            return (
              <Card key={q.id}>
                <CardHeader>
                  <CardTitle className="text-lg">Question {idx + 1}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p>{q.prompt}</p>
                  <div className="space-y-2">
                    {q.options.map((opt) => {
                      const isSelected = selected.includes(opt.id);
                      let borderClass = 'border-transparent';
                      let bgClass = 'bg-gray-50';

                      if (isTerminal) {
                        if (corrects) {
                          // Window closed: show green for correct options, red for wrong selected
                          const isActuallyCorrect = corrects.includes(opt.id);
                          if (isActuallyCorrect) {
                            bgClass = 'bg-green-100';
                            borderClass = 'border-green-500';
                          } else if (isSelected) {
                            bgClass = 'bg-red-100';
                            borderClass = 'border-red-500';
                          }
                        } else {
                          // Window open: highlight their selection
                          if (isSelected) {
                            bgClass = 'bg-blue-100';
                            borderClass = 'border-blue-500';
                          }
                        }
                      } else {
                        // In progress: highlight selection softly
                        if (isSelected) {
                          bgClass = 'bg-blue-50';
                          borderClass = 'border-blue-300';
                        }
                      }

                      return (
                        <div key={opt.id} className={`flex items-center space-x-3 p-3 rounded-lg border ${borderClass} ${bgClass} transition-colors`}>
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
                </CardContent>
              </Card>
            );
          })}
        </div>

        {!isTerminal && (
          <div className="flex justify-end pt-4 border-t">
            <Button onClick={() => submitMutation.mutate()} disabled={submitMutation.isPending} size="lg">
              Submit Attempt
            </Button>
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="p-8 max-w-6xl mx-auto space-y-8">
      <h1 className="text-3xl font-bold tracking-tight">Course Quizzes</h1>

      <div className="grid gap-4 md:grid-cols-2">
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

          return (
            <Card key={quiz.id}>
              <CardHeader className="pb-2">
                <CardTitle className="text-xl flex justify-between">
                  <span>{quiz.title}</span>
                  <span className="text-sm font-normal text-gray-500">{statusText}</span>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-sm text-gray-500 mb-4">Questions: {quiz.max_questions} | Max Grade: {quiz.max_grade}</div>
                <Button 
                  onClick={() => startMutation.mutate(quiz.id)} 
                  disabled={!canTake && !closeAt} // Allow review if closed
                  className="w-full"
                >
                  {closeAt && now > closeAt ? 'Review Attempt' : 'Take / Resume Quiz'}
                </Button>
              </CardContent>
            </Card>
          );
        })}
        {quizzes?.length === 0 && <div className="col-span-2 text-center text-gray-500 py-12">No quizzes available</div>}
      </div>
    </div>
  );
}

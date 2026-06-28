import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { gradesApi } from '@/lib/grades-api';
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent } from '@/components/ui/card';
import { InfoIcon } from 'lucide-react';

export default function StudentGrades() {
  const { id } = useParams();
  const courseId = parseInt(id || '1', 10);

  const { data: scheme, isLoading: loadingScheme } = useQuery({
    queryKey: ['scheme', courseId],
    queryFn: () => gradesApi.getScheme(courseId),
    retry: false,
  });

  const { data: grades, isLoading: loadingGrades } = useQuery({
    queryKey: ['student-grades', courseId],
    queryFn: () => gradesApi.getStudentGrades(courseId),
    retry: false,
  });

  if (loadingScheme || loadingGrades) {
    return (
      <div className="p-8 space-y-10">
        <h1 className="text-4xl font-normal tracking-tight border-l-4 border-primary pl-6">
          My Grades
        </h1>
        <Skeleton className="h-[400px] w-full rounded-lg" />
      </div>
    );
  }

  if (!scheme) {
    return (
      <div className="p-8 space-y-10">
        <h1 className="text-4xl font-normal tracking-tight border-l-4 border-primary pl-6">
          My Grades
        </h1>
        <Card>
          <CardContent className="flex items-center gap-4 pt-6">
            <InfoIcon className="h-6 w-6 text-muted-foreground" strokeWidth={1.5} />
            <div>
              <div className="font-medium text-lg">Not Available</div>
              <div className="text-muted-foreground">
                No grade scheme has been set for this course yet.
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const topLevelComponents = scheme.components.filter(c => !c.parent_id);

  return (
    <div className="p-8 space-y-10">
      <h1 className="text-4xl font-normal tracking-tight border-l-4 border-primary pl-6">
        My Grades
      </h1>

      <section className="bg-card border rounded-lg overflow-hidden shadow-sm">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Component</TableHead>
              <TableHead className="text-right">Weight</TableHead>
              <TableHead className="text-right">Score</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {topLevelComponents.map(c => {
              const compScore = grades?.components.find(gc => gc.component_id === c.id);
              return (
                <TableRow key={c.id}>
                  <TableCell className="text-foreground">{c.name}</TableCell>
                  <TableCell className="text-right font-mono tabular-nums text-muted-foreground">
                    {c.weight}%
                  </TableCell>
                  <TableCell className="text-right">
                    {compScore !== undefined ? (
                      <span className="font-mono tabular-nums font-semibold text-primary">
                        {compScore.score}
                      </span>
                    ) : (
                      <span className="text-sm italic text-muted-foreground">Not published</span>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
          <TableFooter>
            <TableRow className="border-t-2 border-primary/20 bg-muted hover:bg-muted">
              <TableCell
                colSpan={2}
                className="py-6 text-right text-xs font-medium uppercase tracking-wider text-primary"
              >
                Overall
              </TableCell>
              <TableCell className="py-6 text-right">
                {grades?.overall !== null && grades?.overall !== undefined ? (
                  <span className="font-mono tabular-nums text-3xl font-semibold text-primary">
                    {grades.overall}
                  </span>
                ) : (
                  <span className="text-sm font-normal italic text-muted-foreground">Pending</span>
                )}
              </TableCell>
            </TableRow>
          </TableFooter>
        </Table>
      </section>
    </div>
  );
}

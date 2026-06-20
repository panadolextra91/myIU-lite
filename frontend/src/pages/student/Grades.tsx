import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { gradesApi } from '@/lib/grades-api';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
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
    return <div className="p-8"><Skeleton className="h-[400px] w-full" /></div>;
  }

  if (!scheme) {
    return (
      <div className="p-8">
        <h1 className="text-3xl font-bold mb-4">My Grades</h1>
        <Card>
          <CardContent className="flex items-center gap-4 pt-6">
            <InfoIcon className="h-6 w-6 text-muted-foreground" />
            <div>
              <div className="font-semibold text-lg">Not Available</div>
              <div className="text-muted-foreground">No grade scheme has been set for this course yet.</div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const topLevelComponents = scheme.components.filter(c => !c.parent_id);

  return (
    <div className="p-8 space-y-8">
      <h1 className="text-3xl font-bold">My Grades</h1>
      
      <div className="border rounded-lg p-4 bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Component</TableHead>
              <TableHead>Weight</TableHead>
              <TableHead className="text-right">Score</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {topLevelComponents.map(c => {
              const compScore = grades?.components.find(gc => gc.component_id === c.id);
              return (
                <TableRow key={c.id}>
                  <TableCell className="font-medium">{c.name}</TableCell>
                  <TableCell>{c.weight}%</TableCell>
                  <TableCell className="text-right">
                    {compScore !== undefined ? (
                      <span className="font-semibold">{compScore.score}</span>
                    ) : (
                      <span className="text-muted-foreground italic">Not published</span>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
            <TableRow className="bg-muted/50">
              <TableCell className="font-bold text-lg" colSpan={2}>Overall</TableCell>
              <TableCell className="text-right font-bold text-lg">
                {grades?.overall !== null && grades?.overall !== undefined ? (
                  grades.overall
                ) : (
                  <span className="text-muted-foreground text-sm font-normal italic">Pending</span>
                )}
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

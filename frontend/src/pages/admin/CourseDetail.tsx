import { useParams } from 'react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { adminApi } from '@/lib/admin-api';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';

export default function CourseDetail() {
  const { id } = useParams<{ id: string }>();
  const courseId = parseInt(id || '0', 10);
  const queryClient = useQueryClient();

  const { data: course, isLoading: loadingCourse } = useQuery({
    queryKey: ['course', courseId],
    queryFn: () => adminApi.getCourse(courseId),
    enabled: !!courseId,
  });

  const { data: students, isLoading: loadingStudents } = useQuery({
    queryKey: ['course-students', courseId],
    queryFn: () => adminApi.listCourseStudents(courseId),
    enabled: !!courseId,
  });

  const { data: lecturers, isLoading: loadingLecturers } = useQuery({
    queryKey: ['course-lecturers', courseId],
    queryFn: () => adminApi.listCourseLecturers(courseId),
    enabled: !!courseId,
  });

  const removeStudentMutation = useMutation({
    mutationFn: (studentId: number) => adminApi.removeStudent(courseId, studentId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['course-students', courseId] });
      toast.success('Student removed from course');
    },
    onError: () => toast.error('Failed to remove student'),
  });

  const unassignLecturerMutation = useMutation({
    mutationFn: (lecturerId: number) => adminApi.unassignLecturer(courseId, lecturerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['course-lecturers', courseId] });
      toast.success('Lecturer unassigned from course');
    },
    onError: () => toast.error('Failed to unassign lecturer'),
  });

  if (loadingCourse) return <div className="text-muted-foreground">Loading course...</div>;
  if (!course) return <div className="text-muted-foreground">Course not found</div>;

  return (
    <div className="space-y-12">
      <header className="space-y-2">
        <h1 className="text-3xl font-normal tracking-tight">{course.code} - {course.name}</h1>
        <p className="text-sm text-muted-foreground">
          {course.term} • <span className="font-mono tabular-nums">{course.start_date}</span> to <span className="font-mono tabular-nums">{course.end_date}</span>
        </p>
      </header>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="students">Students</TabsTrigger>
          <TabsTrigger value="lecturers">Lecturers</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="pt-6">
          <div className="grid gap-6 sm:grid-cols-2">
            <div className="rounded-lg border bg-card p-6">
              <h4 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Students Enrolled</h4>
              <div className="mt-3 font-mono tabular-nums text-5xl leading-none text-primary">{students?.length || 0}</div>
            </div>
            <div className="rounded-lg border bg-card p-6">
              <h4 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Lecturers</h4>
              <div className="mt-3 font-mono tabular-nums text-5xl leading-none text-primary">{lecturers?.length || 0}</div>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="students" className="pt-6">
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Student ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead className="w-[100px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loadingStudents ? (
                  <TableRow><TableCell colSpan={3} className="text-center text-muted-foreground">Loading...</TableCell></TableRow>
                ) : students?.length === 0 ? (
                  <TableRow><TableCell colSpan={3} className="text-center text-muted-foreground">No students enrolled.</TableCell></TableRow>
                ) : (
                  students?.map((s) => (
                    <TableRow key={s.id}>
                      <TableCell className="font-mono font-medium">{s.username}</TableCell>
                      <TableCell>{s.full_name}</TableCell>
                      <TableCell>
                        <AlertDialog>
                          <AlertDialogTrigger render={<Button variant="ghost" size="sm" className="text-destructive" />}>
                            Remove
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Remove Student?</AlertDialogTitle>
                              <AlertDialogDescription>
                                Are you sure you want to remove {s.username} from {course.code}? This will immediately revoke their access to the course.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction onClick={() => removeStudentMutation.mutate(s.id)} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
                                Remove
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
        </TabsContent>

        <TabsContent value="lecturers" className="pt-6">
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Lecturer ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead className="w-[100px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loadingLecturers ? (
                  <TableRow><TableCell colSpan={3} className="text-center text-muted-foreground">Loading...</TableCell></TableRow>
                ) : lecturers?.length === 0 ? (
                  <TableRow><TableCell colSpan={3} className="text-center text-muted-foreground">No lecturers assigned.</TableCell></TableRow>
                ) : (
                  lecturers?.map((l) => (
                    <TableRow key={l.id}>
                      <TableCell className="font-mono font-medium">{l.username}</TableCell>
                      <TableCell>{l.full_name}</TableCell>
                      <TableCell>
                        <AlertDialog>
                          <AlertDialogTrigger render={<Button variant="ghost" size="sm" className="text-destructive" />}>
                            Unassign
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Unassign Lecturer?</AlertDialogTitle>
                              <AlertDialogDescription>
                                Are you sure you want to unassign {l.username} from {course.code}? They will lose manage access to the course.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction onClick={() => unassignLecturerMutation.mutate(l.id)} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
                                Unassign
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
        </TabsContent>
      </Tabs>
    </div>
  );
}

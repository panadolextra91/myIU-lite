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

  if (loadingCourse) return <div>Loading course...</div>;
  if (!course) return <div>Course not found</div>;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">{course.code} - {course.name}</h2>
        <p className="text-muted-foreground">{course.term} • {course.start_date} to {course.end_date}</p>
      </div>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="students">Students</TabsTrigger>
          <TabsTrigger value="lecturers">Lecturers</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="space-y-4 pt-4">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <div className="rounded-xl border bg-card text-card-foreground shadow-sm p-6">
              <h3 className="tracking-tight text-sm font-medium text-muted-foreground">Students Enrolled</h3>
              <div className="text-2xl font-bold">{students?.length || 0}</div>
            </div>
            <div className="rounded-xl border bg-card text-card-foreground shadow-sm p-6">
              <h3 className="tracking-tight text-sm font-medium text-muted-foreground">Lecturers</h3>
              <div className="text-2xl font-bold">{lecturers?.length || 0}</div>
            </div>
          </div>
        </TabsContent>
        <TabsContent value="students" className="pt-4">
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
                  <TableRow><TableCell colSpan={3} className="text-center">Loading...</TableCell></TableRow>
                ) : students?.length === 0 ? (
                  <TableRow><TableCell colSpan={3} className="text-center">No students enrolled.</TableCell></TableRow>
                ) : (
                  students?.map((s) => (
                    <TableRow key={s.id}>
                      <TableCell className="font-medium">{s.username}</TableCell>
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
        <TabsContent value="lecturers" className="pt-4">
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
                  <TableRow><TableCell colSpan={3} className="text-center">Loading...</TableCell></TableRow>
                ) : lecturers?.length === 0 ? (
                  <TableRow><TableCell colSpan={3} className="text-center">No lecturers assigned.</TableCell></TableRow>
                ) : (
                  lecturers?.map((l) => (
                    <TableRow key={l.id}>
                      <TableCell className="font-medium">{l.username}</TableCell>
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

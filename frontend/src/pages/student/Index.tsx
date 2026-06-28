import { FileText, ClipboardList, GraduationCap, Megaphone, Send } from 'lucide-react';
import { DashboardLanding, type TocItem } from '@/components/DashboardLanding';

const items: TocItem[] = [
  { icon: FileText, label: 'Assignments', description: 'Submit and track coursework.', to: '/student/assignments' },
  { icon: ClipboardList, label: 'Quizzes', description: 'Take and review quizzes.', to: '/student/quizzes' },
  { icon: GraduationCap, label: 'Grades', description: 'View your published grades.' },
  { icon: Megaphone, label: 'Announcements', description: 'Course updates from lecturers.' },
  { icon: Send, label: 'Requests', description: 'Message your lecturer.' },
];

export default function StudentIndex() {
  return (
    <DashboardLanding
      title="Student"
      subtitle="Welcome back. Choose where to continue."
      items={items}
    />
  );
}

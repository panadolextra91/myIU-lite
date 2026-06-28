import { FileText, ClipboardList, BookOpen, Megaphone, Inbox } from 'lucide-react';
import { DashboardLanding, type TocItem } from '@/components/DashboardLanding';

const items: TocItem[] = [
  { icon: FileText, label: 'Assignments', description: 'Create, grade and finalize.', to: '/lecturer/assignments' },
  { icon: Inbox, label: 'Request Inbox', description: 'Reply to student requests.', to: '/lecturer/requests' },
  { icon: ClipboardList, label: 'Quizzes', description: 'Build and author quizzes.' },
  { icon: BookOpen, label: 'Gradebook', description: 'Manage the grade scheme and publish.' },
  { icon: Megaphone, label: 'Announcements', description: 'Post to your students.' },
];

export default function LecturerIndex() {
  return (
    <DashboardLanding
      title="Lecturer"
      subtitle="Welcome back. Choose where to continue."
      items={items}
    />
  );
}

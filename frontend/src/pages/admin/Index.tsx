import { Users, BookOpen, GraduationCap, UserPlus, ShieldAlert } from 'lucide-react';
import { DashboardLanding, type TocItem } from '@/components/DashboardLanding';

const items: TocItem[] = [
  { icon: Users, label: 'Accounts', description: 'Manage student and lecturer accounts.', to: '/admin/accounts' },
  { icon: BookOpen, label: 'Courses', description: 'Manage courses and lifecycles.', to: '/admin/courses' },
  { icon: GraduationCap, label: 'Student Enrollment', description: 'Bulk-enroll students via CSV.', to: '/admin/enrollment' },
  { icon: UserPlus, label: 'Lecturer Assignment', description: 'Assign lecturers via CSV.', to: '/admin/lecturers' },
  { icon: ShieldAlert, label: 'Audit Logs', description: 'Review administrative actions.', to: '/admin/audit' },
];

export default function AdminIndex() {
  return (
    <DashboardLanding
      title="Administration"
      subtitle="Welcome back. Choose an area to manage."
      items={items}
    />
  );
}

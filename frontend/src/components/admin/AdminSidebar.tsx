import { Link, useLocation } from 'react-router';
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar';
import { LayoutDashboard, Users, BookOpen, UserPlus, GraduationCap, ShieldAlert } from 'lucide-react';

const navItems = [
  { title: 'Dashboard', url: '/admin', icon: LayoutDashboard },
  { title: 'Accounts', url: '/admin/accounts', icon: Users, group: 'User Management' },
  { title: 'Courses', url: '/admin/courses', icon: BookOpen, group: 'Academic Management' },
  { title: 'Student Enrollment', url: '/admin/enrollment', icon: GraduationCap, group: 'Academic Management' },
  { title: 'Lecturer Assignment', url: '/admin/lecturers', icon: UserPlus, group: 'Academic Management' },
  { title: 'Audit Logs', url: '/admin/audit', icon: ShieldAlert, group: 'System' },
];

export function AdminSidebar() {
  const location = useLocation();

  const groups = navItems.reduce((acc, item) => {
    const group = item.group || 'General';
    if (!acc[group]) acc[group] = [];
    acc[group].push(item);
    return acc;
  }, {} as Record<string, typeof navItems>);

  return (
    <Sidebar>
      <SidebarContent>
        {Object.entries(groups).map(([group, items]) => (
          <SidebarGroup key={group}>
            {group !== 'General' && <SidebarGroupLabel>{group}</SidebarGroupLabel>}
            <SidebarGroupContent>
              <SidebarMenu>
                {items.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton 
                    render={<Link to={item.url} />}
                    isActive={location.pathname.startsWith(item.url)}
                    tooltip={item.title}
                  >
                    <item.icon />
                    <span>{item.title}</span>
                  </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
    </Sidebar>
  );
}

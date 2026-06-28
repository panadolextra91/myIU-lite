import { Link } from 'react-router';

export type TocItem = {
  icon: React.ComponentType<{ className?: string; strokeWidth?: number }>;
  label: string;
  description: string;
  to?: string;
};

/**
 * Editorial "table of contents" landing used by the three role dashboards.
 * Sections that route to a course-scoped page have no `to` (the app has no
 * course picker yet) and render as non-linked index rows.
 */
export function DashboardLanding({
  title,
  subtitle,
  items,
}: {
  title: string;
  subtitle: string;
  items: TocItem[];
}) {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10">
      <h1 className="font-heading text-4xl font-normal tracking-tight">{title}</h1>
      <p className="mt-1 text-muted-foreground">{subtitle}</p>

      <div className="mt-10 border-y border-border divide-y divide-border">
        {items.map((item) => {
          const Icon = item.icon;
          const inner = (
            <div className="flex items-start gap-4 py-5">
              <Icon className="mt-1 size-5 shrink-0 text-muted-foreground" strokeWidth={1.5} />
              <div className="space-y-0.5">
                <div className="font-heading text-xl leading-snug">{item.label}</div>
                <div className="text-sm text-muted-foreground">{item.description}</div>
              </div>
            </div>
          );

          return item.to ? (
            <Link
              key={item.label}
              to={item.to}
              className="block px-2 -mx-2 rounded-md transition-colors hover:bg-muted/50"
            >
              {inner}
            </Link>
          ) : (
            // ponytail: course-scoped sections need a selected course (existing nav gap); index row until a course picker exists.
            <div key={item.label} className="px-2 -mx-2 opacity-70">
              {inner}
            </div>
          );
        })}
      </div>
    </div>
  );
}

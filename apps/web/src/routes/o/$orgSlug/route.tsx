import { createFileRoute, notFound, Outlet } from "@tanstack/react-router";
import { AppShell } from "@/components/shell/app-shell";
import { meQueryOptions } from "@/features/auth/me";

const SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

export const Route = createFileRoute("/o/$orgSlug")({
  beforeLoad: async ({ params, context }) => {
    if (!SLUG.test(params.orgSlug) || params.orgSlug.length > 63) throw notFound();
    const me = await context.queryClient.ensureQueryData(meQueryOptions());
    const membership = me.memberships.find((m) => m.slug === params.orgSlug);
    if (!membership) throw notFound();
    return { membership, me };
  },
  component: OrgLayout,
});

function OrgLayout() {
  const { orgSlug } = Route.useParams();
  const { membership, me } = Route.useRouteContext();
  return (
    <AppShell
      orgSlug={orgSlug}
      membership={membership}
      memberships={me.memberships}
      displayName={me.user.displayName || me.user.email || "Account"}
    >
      <Outlet />
    </AppShell>
  );
}

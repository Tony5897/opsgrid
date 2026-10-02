import { createFileRoute, notFound, Outlet } from "@tanstack/react-router";
import { AppShell } from "@/components/shell/app-shell";

const SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

export const Route = createFileRoute("/o/$orgSlug")({
  // Reject malformed slugs before any data loading (Gate 1 resolves the slug
  // against the caller's memberships and returns 404 for non-members).
  beforeLoad: ({ params }) => {
    if (!SLUG.test(params.orgSlug) || params.orgSlug.length > 63) throw notFound();
  },
  component: OrgLayout,
});

function OrgLayout() {
  const { orgSlug } = Route.useParams();
  return (
    <AppShell orgSlug={orgSlug}>
      <Outlet />
    </AppShell>
  );
}

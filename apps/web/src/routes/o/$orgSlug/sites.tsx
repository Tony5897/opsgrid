import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/sites")({
  component: SitesPage,
});

function SitesPage() {
  return <PlaceholderPage path="/o/$orgSlug/sites" />;
}

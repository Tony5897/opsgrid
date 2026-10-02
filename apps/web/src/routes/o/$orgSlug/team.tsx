import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/team")({
  component: TeamPage,
});

function TeamPage() {
  return <PlaceholderPage path="/o/$orgSlug/team" />;
}

import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/schedule")({
  component: SchedulePage,
});

function SchedulePage() {
  return <PlaceholderPage path="/o/$orgSlug/schedule" />;
}

import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/")({
  component: OverviewPage,
});

function OverviewPage() {
  return <PlaceholderPage path="/o/$orgSlug" />;
}

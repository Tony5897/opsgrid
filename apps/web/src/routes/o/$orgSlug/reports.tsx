import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/reports")({
  component: ReportsPage,
});

function ReportsPage() {
  return <PlaceholderPage path="/o/$orgSlug/reports" />;
}

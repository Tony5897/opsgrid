import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/work-orders")({
  component: WorkOrdersPage,
});

function WorkOrdersPage() {
  return <PlaceholderPage path="/o/$orgSlug/work-orders" />;
}

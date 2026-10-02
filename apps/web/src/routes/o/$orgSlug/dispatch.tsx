import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/dispatch")({
  component: DispatchPage,
});

function DispatchPage() {
  return <PlaceholderPage path="/o/$orgSlug/dispatch" />;
}

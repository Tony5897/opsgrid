import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/audit")({
  component: AuditPage,
});

function AuditPage() {
  return <PlaceholderPage path="/o/$orgSlug/audit" />;
}

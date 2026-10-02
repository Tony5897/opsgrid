import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/customers")({
  component: CustomersPage,
});

function CustomersPage() {
  return <PlaceholderPage path="/o/$orgSlug/customers" />;
}

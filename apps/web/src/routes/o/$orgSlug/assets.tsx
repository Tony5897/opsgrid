import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/assets")({
  component: AssetsPage,
});

function AssetsPage() {
  return <PlaceholderPage path="/o/$orgSlug/assets" />;
}

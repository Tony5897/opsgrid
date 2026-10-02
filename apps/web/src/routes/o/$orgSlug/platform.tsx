import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/platform")({
  component: PlatformPage,
});

function PlatformPage() {
  return <PlaceholderPage path="/o/$orgSlug/platform" />;
}

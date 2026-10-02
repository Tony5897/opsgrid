import { createFileRoute } from "@tanstack/react-router";
import { PlaceholderPage } from "@/components/placeholder-page";

export const Route = createFileRoute("/o/$orgSlug/settings")({
  component: SettingsPage,
});

function SettingsPage() {
  return <PlaceholderPage path="/o/$orgSlug/settings" />;
}

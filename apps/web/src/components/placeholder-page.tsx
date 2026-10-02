import { ConstructionIcon } from "lucide-react";
import { ALL_NAV_ITEMS, type NavPath } from "@/components/shell/nav";
import { EmptyState } from "./empty-state";
import { PageHeader } from "./page-header";

/** Honest placeholder for areas delivered by later gates. */
export function PlaceholderPage({ path }: { path: NavPath }) {
  const item = ALL_NAV_ITEMS.find((i) => i.to === path);
  if (!item) throw new Error(`no nav item for ${path}`);
  return (
    <>
      <PageHeader title={item.label} description={item.summary} />
      <EmptyState
        icon={item.icon ?? ConstructionIcon}
        title={`Arrives in Gate ${item.gate.slice(1)}`}
        description="This area is planned and specified in the implementation plan. Its data, permissions and tests land together when its gate closes."
      />
    </>
  );
}

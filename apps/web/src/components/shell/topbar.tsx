import type { MeMembership } from "@opsgrid/api-client";
import { SearchIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Kbd } from "@/components/ui/kbd";
import { Tooltip } from "@/components/ui/tooltip";
import { useOnlineStatus } from "@/lib/use-online-status";
import { ConnectionStatus } from "./connection-status";
import { MobileNav } from "./mobile-nav";
import { OrgSwitcher } from "./org-switcher";
import { UserMenu } from "./user-menu";

export function Topbar({
  orgSlug,
  memberships,
  displayName,
}: {
  orgSlug: string;
  memberships: MeMembership[];
  displayName: string;
}) {
  const online = useOnlineStatus();
  const permissions = memberships.find((m) => m.slug === orgSlug)?.permissions ?? [];
  return (
    <header className="sticky top-0 z-30 flex h-topbar shrink-0 items-center gap-2 border-b border-border bg-surface-1/90 px-3 backdrop-blur supports-[backdrop-filter]:bg-surface-1/75 sm:px-4">
      <MobileNav orgSlug={orgSlug} permissions={permissions} />
      <div className="flex min-w-0 items-center gap-2">
        <OrgSwitcher orgSlug={orgSlug} memberships={memberships} />
      </div>

      <div className="ml-auto flex items-center gap-1 sm:gap-2">
        <Tooltip content="Search and commands arrive in Gate 2">
          <Button
            variant="secondary"
            size="sm"
            className="w-9 px-0 text-text-secondary sm:w-56 sm:justify-start sm:px-3"
            aria-disabled="true"
            aria-label="Search (coming in Gate 2)"
          >
            <SearchIcon aria-hidden />
            <span className="hidden sm:inline">Search…</span>
            <span className="ml-auto hidden gap-0.5 sm:flex">
              <Kbd>⌘</Kbd>
              <Kbd>K</Kbd>
            </span>
          </Button>
        </Tooltip>
        <ConnectionStatus state={online ? "connected" : "offline"} />
        <UserMenu displayName={displayName} />
      </div>
    </header>
  );
}

import type { MeMembership } from "@opsgrid/api-client";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { ChevronsUpDownIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function OrgSwitcher({
  orgSlug,
  memberships,
}: {
  orgSlug: string;
  memberships: MeMembership[];
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const current = memberships.find((m) => m.slug === orgSlug);
  const label = current?.name ?? orgSlug;

  async function switchTo(slug: string) {
    if (slug === orgSlug) return;
    // Drop org-scoped caches so the next org does not see stale data.
    await queryClient.cancelQueries();
    queryClient.removeQueries({
      predicate: (q) => q.queryKey.some((k) => k === orgSlug || k === `org:${orgSlug}`),
    });
    void navigate({ to: "/o/$orgSlug", params: { orgSlug: slug } });
  }

  if (memberships.length <= 1) {
    return (
      <span className="truncate text-sm font-semibold text-text-primary" title={label}>
        {label}
      </span>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="max-w-[14rem] gap-1 px-2 font-semibold"
          aria-label="Switch organization"
        >
          <span className="truncate">{label}</span>
          <ChevronsUpDownIcon aria-hidden className="size-3.5 shrink-0 text-text-subtle" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-64">
        <DropdownMenuLabel>Organizations</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {memberships.map((m) => (
          <DropdownMenuItem
            key={m.organizationId}
            onSelect={() => void switchTo(m.slug)}
            data-active={m.slug === orgSlug ? "true" : undefined}
            className={m.slug === orgSlug ? "bg-accent-subtle text-accent-subtle-fg" : undefined}
          >
            <span className="flex min-w-0 flex-col">
              <span className="truncate font-medium">{m.name}</span>
              <span className="truncate text-xs text-text-secondary capitalize">{m.role}</span>
            </span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

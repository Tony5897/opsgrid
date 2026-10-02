import { Link } from "@tanstack/react-router";
import { PanelLeftCloseIcon, PanelLeftOpenIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { setPreferences, usePreferences } from "@/lib/preferences";
import { cn } from "@/lib/utils";
import { NAV } from "./nav";

interface SidebarNavProps {
  orgSlug: string;
  collapsed: boolean;
  onNavigate?: () => void;
}

/** Primary navigation list, shared by the desktop rail and the mobile sheet. */
export function SidebarNav({ orgSlug, collapsed, onNavigate }: SidebarNavProps) {
  return (
    <nav aria-label="Primary" className="flex-1 overflow-y-auto px-2 py-3">
      {NAV.map((section) => (
        <div key={section.label} className="mb-4 last:mb-0">
          <h2
            className={cn(
              "px-2 pb-1 text-[11px] font-semibold tracking-wider text-text-subtle uppercase",
              collapsed && "sr-only",
            )}
          >
            {section.label}
          </h2>
          <ul className="space-y-0.5">
            {section.items.map((item) => {
              const Icon = item.icon;
              const link = (
                <Link
                  to={item.to}
                  params={{ orgSlug }}
                  activeOptions={{ exact: item.to === "/o/$orgSlug" }}
                  onClick={onNavigate}
                  className={cn(
                    "group flex h-9 items-center gap-3 rounded-md px-2 text-sm font-medium text-text-secondary",
                    "transition-colors duration-(--duration-fast) hover:bg-surface-3 hover:text-text-primary",
                    "aria-[current=page]:bg-accent-subtle aria-[current=page]:text-accent-subtle-fg",
                    collapsed && "justify-center px-0",
                  )}
                >
                  <Icon aria-hidden className="size-[18px] shrink-0" />
                  <span className={cn("truncate", collapsed && "sr-only")}>{item.label}</span>
                </Link>
              );
              return (
                <li key={item.to}>
                  {collapsed ? (
                    <Tooltip content={item.label} side="right">
                      {link}
                    </Tooltip>
                  ) : (
                    link
                  )}
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </nav>
  );
}

export function Brand({ collapsed }: { collapsed: boolean }) {
  return (
    <div
      className={cn("flex h-topbar items-center gap-2.5 px-4", collapsed && "justify-center px-0")}
    >
      <img src="/favicon.svg" alt="" className="size-7" />
      <span className={cn("text-md font-semibold tracking-tight", collapsed && "sr-only")}>
        OpsGrid
      </span>
    </div>
  );
}

/** Desktop sidebar (md and up). Collapses to an icon rail; state persists per device. */
export function Sidebar({ orgSlug }: { orgSlug: string }) {
  const { sidebarCollapsed: collapsed } = usePreferences();
  return (
    <aside
      className={cn(
        "sticky top-0 hidden h-dvh shrink-0 flex-col border-r border-border bg-surface-1 md:flex",
        "transition-[width] duration-(--duration-base) ease-out",
        collapsed ? "w-sidebar-collapsed" : "w-sidebar",
      )}
    >
      <Brand collapsed={collapsed} />
      <SidebarNav orgSlug={orgSlug} collapsed={collapsed} />
      <div className={cn("border-t border-border p-2", collapsed && "flex justify-center")}>
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          aria-expanded={!collapsed}
          onClick={() => setPreferences({ sidebarCollapsed: !collapsed })}
        >
          {collapsed ? <PanelLeftOpenIcon aria-hidden /> : <PanelLeftCloseIcon aria-hidden />}
        </Button>
      </div>
    </aside>
  );
}

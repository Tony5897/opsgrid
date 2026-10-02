import {
  ActivityIcon,
  BoxIcon,
  Building2Icon,
  CalendarRangeIcon,
  ClipboardListIcon,
  FileBarChartIcon,
  LayoutDashboardIcon,
  type LucideIcon,
  MapPinIcon,
  RadioTowerIcon,
  ScrollTextIcon,
  SettingsIcon,
  UsersIcon,
} from "lucide-react";

export type NavPath =
  | "/o/$orgSlug"
  | "/o/$orgSlug/dispatch"
  | "/o/$orgSlug/schedule"
  | "/o/$orgSlug/work-orders"
  | "/o/$orgSlug/customers"
  | "/o/$orgSlug/sites"
  | "/o/$orgSlug/assets"
  | "/o/$orgSlug/reports"
  | "/o/$orgSlug/audit"
  | "/o/$orgSlug/team"
  | "/o/$orgSlug/settings"
  | "/o/$orgSlug/platform";

export interface NavItem {
  /** Typed route path; $orgSlug is filled from the current route. */
  to: NavPath;
  label: string;
  icon: LucideIcon;
  /** Gate that delivers this area (shown on placeholders until then). */
  gate: string;
  summary: string;
}

export interface NavSection {
  label: string;
  items: NavItem[];
}

export const NAV: NavSection[] = [
  {
    label: "Operations",
    items: [
      {
        to: "/o/$orgSlug",
        label: "Overview",
        icon: LayoutDashboardIcon,
        gate: "G2",
        summary: "Today's workload, backlog and technician utilization at a glance.",
      },
      {
        to: "/o/$orgSlug/dispatch",
        label: "Dispatch",
        icon: RadioTowerIcon,
        gate: "G3",
        summary:
          "Live board: drag unassigned work onto technician lanes, with database-enforced conflict prevention.",
      },
      {
        to: "/o/$orgSlug/schedule",
        label: "Schedule",
        icon: CalendarRangeIcon,
        gate: "G3",
        summary: "Week view of every technician's assignments in each location's timezone.",
      },
      {
        to: "/o/$orgSlug/work-orders",
        label: "Work orders",
        icon: ClipboardListIcon,
        gate: "G2",
        summary: "Filterable, URL-addressable list of all work with saved views.",
      },
    ],
  },
  {
    label: "Records",
    items: [
      {
        to: "/o/$orgSlug/customers",
        label: "Customers",
        icon: Building2Icon,
        gate: "G2",
        summary: "Customer companies and their contacts.",
      },
      {
        to: "/o/$orgSlug/sites",
        label: "Sites",
        icon: MapPinIcon,
        gate: "G2",
        summary: "Physical customer locations, each with its own timezone.",
      },
      {
        to: "/o/$orgSlug/assets",
        label: "Assets",
        icon: BoxIcon,
        gate: "G2",
        summary: "Equipment installed at sites, with full service history.",
      },
    ],
  },
  {
    label: "Insight",
    items: [
      {
        to: "/o/$orgSlug/reports",
        label: "Reports",
        icon: FileBarChartIcon,
        gate: "G7",
        summary: "Asynchronous CSV and PDF exports delivered when ready.",
      },
      {
        to: "/o/$orgSlug/audit",
        label: "Audit log",
        icon: ScrollTextIcon,
        gate: "G2",
        summary:
          "Append-only record of every sensitive change: who, what, when and the request that did it.",
      },
    ],
  },
  {
    label: "Administration",
    items: [
      {
        to: "/o/$orgSlug/team",
        label: "Team",
        icon: UsersIcon,
        gate: "G1",
        summary: "Members, roles and the permission matrix.",
      },
      {
        to: "/o/$orgSlug/settings",
        label: "Settings",
        icon: SettingsIcon,
        gate: "G1",
        summary: "Organization profile, locations and preferences.",
      },
      {
        to: "/o/$orgSlug/platform",
        label: "Platform health",
        icon: ActivityIcon,
        gate: "G6",
        summary: "Queues, outbox lag, dead letters and realtime connections.",
      },
    ],
  },
];

export const ALL_NAV_ITEMS: NavItem[] = NAV.flatMap((s) => s.items);

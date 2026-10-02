import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowRightIcon } from "lucide-react";
import { useEffect } from "react";

export const Route = createFileRoute("/")({
  component: Landing,
});

// Demo tenants from the architecture overview. In Gate 1 this page becomes
// the post-login organization picker backed by /v1/me memberships.
const DEMO_ORGS = [
  {
    slug: "cascade-facilities",
    name: "Cascade Facilities",
    locations: "Portland · Salem · Eugene",
  },
  { slug: "northstar-mechanical", name: "Northstar Mechanical", locations: "Seattle · Tacoma" },
];

function Landing() {
  useEffect(() => {
    document.title = "OpsGrid";
  }, []);
  return (
    <main
      id="main-content"
      className="flex min-h-dvh items-center justify-center bg-canvas px-4 py-16"
    >
      <div className="w-full max-w-md space-y-8">
        <div className="space-y-3 text-center">
          <img src="/favicon.svg" alt="" className="mx-auto size-10" />
          <h1 className="text-2xl font-semibold tracking-tight">OpsGrid</h1>
          <p className="text-sm text-text-secondary">
            Real-time field operations. Sign-in with your organization arrives in Gate 1. For now,
            preview the console shell for a demo tenant.
          </p>
        </div>
        <ul className="space-y-2">
          {DEMO_ORGS.map((org) => (
            <li key={org.slug}>
              <Link
                to="/o/$orgSlug"
                params={{ orgSlug: org.slug }}
                className="group flex items-center justify-between rounded-lg border border-border bg-surface-1 px-4 py-3 shadow-sm transition-colors hover:border-border-strong hover:bg-surface-2"
              >
                <span>
                  <span className="block text-base font-medium">{org.name}</span>
                  <span className="block text-sm text-text-secondary">{org.locations}</span>
                </span>
                <ArrowRightIcon
                  aria-hidden
                  className="size-4 text-text-subtle transition-transform group-hover:translate-x-0.5"
                />
              </Link>
            </li>
          ))}
        </ul>
      </div>
    </main>
  );
}

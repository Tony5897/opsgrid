import type { ReactNode } from "react";
import { useRouteFocus } from "@/lib/use-route-focus";
import { Sidebar } from "./sidebar";
import { Topbar } from "./topbar";

export function AppShell({ orgSlug, children }: { orgSlug: string; children: ReactNode }) {
  useRouteFocus();
  return (
    <div className="flex min-h-dvh bg-canvas">
      <a href="#main-content" className="skip-link">
        Skip to main content
      </a>
      <Sidebar orgSlug={orgSlug} />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar orgSlug={orgSlug} />
        <main
          id="main-content"
          tabIndex={-1}
          className="flex-1 px-4 py-6 outline-none sm:px-6 lg:px-8"
        >
          <div className="mx-auto w-full max-w-7xl">{children}</div>
        </main>
      </div>
    </div>
  );
}

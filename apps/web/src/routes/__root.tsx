import type { QueryClient } from "@tanstack/react-query";
import {
  createRootRouteWithContext,
  type ErrorComponentProps,
  Link,
  Outlet,
} from "@tanstack/react-router";
import { CompassIcon, TriangleAlertIcon } from "lucide-react";
import { lazy, Suspense } from "react";
import { EmptyState } from "@/components/empty-state";
import { Button } from "@/components/ui/button";

export interface RouterContext {
  queryClient: QueryClient;
}

// Devtools are code-split and only ever loaded in development.
const Devtools = import.meta.env.DEV
  ? lazy(() => import("@/app/devtools").then((m) => ({ default: m.Devtools })))
  : () => null;

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
  notFoundComponent: NotFound,
  errorComponent: RootError,
});

function RootLayout() {
  return (
    <>
      <Outlet />
      <Suspense>
        <Devtools />
      </Suspense>
    </>
  );
}

function NotFound() {
  return (
    <main id="main-content" className="mx-auto max-w-xl px-6 py-24">
      <EmptyState
        icon={CompassIcon}
        title="Page not found"
        description="The address may be mistyped, or the page may have moved."
        action={
          <Button asChild variant="primary" size="sm">
            <Link to="/">Go to start</Link>
          </Button>
        }
      />
    </main>
  );
}

function RootError({ error, reset }: ErrorComponentProps) {
  return (
    <main id="main-content" className="mx-auto max-w-xl px-6 py-24">
      <EmptyState
        icon={TriangleAlertIcon}
        title="Something went wrong"
        description={
          import.meta.env.DEV && error instanceof Error
            ? error.message
            : "An unexpected error occurred. Try again, or reload the page."
        }
        action={
          <Button variant="primary" size="sm" onClick={reset}>
            Try again
          </Button>
        }
      />
    </main>
  );
}

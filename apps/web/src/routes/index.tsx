import { ApiError, startLogin, type Me } from "@opsgrid/api-client";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowRightIcon } from "lucide-react";
import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { meQueryOptions } from "@/features/auth/me";

export const Route = createFileRoute("/")({
  component: Landing,
});

function Landing() {
  useEffect(() => {
    document.title = "OpsGrid";
  }, []);

  const me = useQuery({
    ...meQueryOptions({ redirectOnAuth: false }),
    retry: false,
  });

  if (me.isPending) {
    return (
      <main
        id="main-content"
        className="flex min-h-dvh items-center justify-center bg-canvas px-4 py-16"
      >
        <p className="text-sm text-text-secondary">Loading…</p>
      </main>
    );
  }

  const authed = me.isSuccess;
  const unauthorized =
    me.isError && me.error instanceof ApiError && me.error.code === "AUTHENTICATION_REQUIRED";

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
            {authed
              ? `Signed in as ${me.data.user.displayName || me.data.user.email || "you"}. Choose an organization to continue.`
              : "Real-time field operations. Sign in with your organization account to open the console."}
          </p>
        </div>

        {authed ? <MembershipList me={me.data} /> : null}

        {!authed ? (
          <div className="flex flex-col items-center gap-3">
            <Button
              variant="primary"
              size="md"
              onClick={() => startLogin("/")}
              type="button"
            >
              Sign in
            </Button>
            {me.isError && !unauthorized ? (
              <p className="text-sm text-text-secondary">
                Could not reach the API. Is the stack running?
              </p>
            ) : null}
          </div>
        ) : null}
      </div>
    </main>
  );
}

function MembershipList({ me }: { me: Me }) {
  if (me.memberships.length === 0) {
    return (
      <p className="rounded-lg border border-border bg-surface-1 px-4 py-3 text-sm text-text-secondary">
        You are signed in but not a member of any organization yet.
      </p>
    );
  }
  return (
    <ul className="space-y-2">
      {me.memberships.map((org) => (
        <li key={org.organizationId}>
          <Link
            to="/o/$orgSlug"
            params={{ orgSlug: org.slug }}
            className="group flex items-center justify-between rounded-lg border border-border bg-surface-1 px-4 py-3 shadow-sm transition-colors hover:border-border-strong hover:bg-surface-2"
          >
            <span>
              <span className="block text-base font-medium">{org.name}</span>
              <span className="block text-sm text-text-secondary capitalize">{org.role}</span>
            </span>
            <ArrowRightIcon
              aria-hidden
              className="size-4 text-text-subtle transition-transform group-hover:translate-x-0.5"
            />
          </Link>
        </li>
      ))}
    </ul>
  );
}

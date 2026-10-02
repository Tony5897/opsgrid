import { MenuIcon, XIcon } from "lucide-react";
import { Dialog } from "radix-ui";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Brand, SidebarNav } from "./sidebar";

/** Below md the sidebar becomes a modal sheet (focus-trapped, Esc closes). */
export function MobileNav({ orgSlug }: { orgSlug: string }) {
  const [open, setOpen] = useState(false);
  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <Button variant="ghost" size="icon-sm" className="md:hidden" aria-label="Open navigation">
          <MenuIcon aria-hidden />
        </Button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-(--surface-overlay) data-[state=open]:animate-in data-[state=open]:fade-in-0 md:hidden" />
        <Dialog.Content className="fixed inset-y-0 left-0 z-50 flex w-72 max-w-[85vw] flex-col bg-surface-1 shadow-lg data-[state=open]:animate-in data-[state=open]:slide-in-from-left md:hidden">
          <Dialog.Title className="sr-only">Navigation</Dialog.Title>
          <Dialog.Description className="sr-only">Main sections of OpsGrid</Dialog.Description>
          <div className="flex items-center justify-between pr-2">
            <Brand collapsed={false} />
            <Dialog.Close asChild>
              <Button variant="ghost" size="icon-sm" aria-label="Close navigation">
                <XIcon aria-hidden />
              </Button>
            </Dialog.Close>
          </div>
          <SidebarNav orgSlug={orgSlug} collapsed={false} onNavigate={() => setOpen(false)} />
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

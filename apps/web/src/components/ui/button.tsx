import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "radix-ui";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

export const buttonVariants = cva(
  [
    "inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium select-none",
    "transition-colors duration-(--duration-fast) ease-out",
    "disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50",
    "[&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  ],
  {
    variants: {
      variant: {
        primary: "bg-accent text-accent-fg shadow-sm hover:bg-accent-hover",
        secondary:
          "border border-border bg-surface-1 text-text-primary shadow-sm hover:bg-surface-3",
        ghost: "text-text-secondary hover:bg-surface-3 hover:text-text-primary",
        danger: "bg-danger text-destructive-foreground shadow-sm hover:opacity-90",
        link: "h-auto p-0 text-text-link underline-offset-4 hover:underline",
      },
      size: {
        sm: "h-8 px-3 text-sm",
        md: "h-control px-4 text-base",
        icon: "size-control",
        "icon-sm": "size-8",
      },
    },
    defaultVariants: { variant: "secondary", size: "md" },
  },
);

export interface ButtonProps extends ComponentProps<"button">, VariantProps<typeof buttonVariants> {
  /** Render the child element (e.g. a router Link) with button styling. */
  asChild?: boolean;
}

export function Button({ className, variant, size, asChild = false, type, ...props }: ButtonProps) {
  const Comp = asChild ? Slot.Root : "button";
  return (
    <Comp
      data-slot="button"
      className={cn(buttonVariants({ variant, size }), className)}
      // Default to type="button" so buttons never submit forms by accident.
      {...(!asChild && { type: type ?? "button" })}
      {...props}
    />
  );
}

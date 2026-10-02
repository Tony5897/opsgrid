import { LaptopIcon, MoonIcon, Rows3Icon, Rows4Icon, SunIcon, UserIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  type Density,
  setPreferences,
  type ThemePreference,
  usePreferences,
} from "@/lib/preferences";

export function UserMenu() {
  const { theme, density } = usePreferences();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon-sm" aria-label="Account and display settings">
          <UserIcon aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel>Theme</DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={theme}
          onValueChange={(v) => setPreferences({ theme: v as ThemePreference })}
        >
          <DropdownMenuRadioItem value="system">
            <LaptopIcon aria-hidden /> System
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="light">
            <SunIcon aria-hidden /> Light
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="dark">
            <MoonIcon aria-hidden /> Dark
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>Density</DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={density}
          onValueChange={(v) => setPreferences({ density: v as Density })}
        >
          <DropdownMenuRadioItem value="comfortable">
            <Rows3Icon aria-hidden /> Comfortable
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="compact">
            <Rows4Icon aria-hidden /> Compact
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

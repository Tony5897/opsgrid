import type { Meta, StoryObj } from "@storybook/react-vite";

const SURFACES = [
  "--surface-canvas",
  "--surface-1",
  "--surface-2",
  "--surface-3",
  "--surface-inverse",
];
const TEXT = ["--text-primary", "--text-secondary", "--text-subtle", "--text-link"];
const FEEDBACK = [
  "--accent",
  "--danger",
  "--success",
  "--warning",
  "--focus-ring",
  "--border-strong",
];

function Swatch({ token }: { token: string }) {
  return (
    <div className="flex items-center gap-3">
      <span
        className="size-10 rounded-md border border-border"
        style={{ background: `var(${token})` }}
      />
      <code className="font-mono text-xs text-text-secondary">{token}</code>
    </div>
  );
}

function Palette() {
  return (
    <div className="grid gap-8 md:grid-cols-3">
      {[
        ["Surfaces", SURFACES],
        ["Text", TEXT],
        ["Feedback & UI", FEEDBACK],
      ].map(([title, tokens]) => (
        <section key={title as string} className="space-y-3">
          <h2 className="text-sm font-semibold">{title}</h2>
          {(tokens as string[]).map((t) => (
            <Swatch key={t} token={t} />
          ))}
        </section>
      ))}
    </div>
  );
}

const meta = { title: "Foundations/Color tokens", component: Palette } satisfies Meta<
  typeof Palette
>;
export default meta;
export const Palette_: StoryObj<typeof meta> = { name: "Palette" };

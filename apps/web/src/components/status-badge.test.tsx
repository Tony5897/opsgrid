import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-react";
import { STATUS_META, WORK_ORDER_STATUSES } from "@/features/work-orders/status";
import { StatusBadge } from "./status-badge";

describe("StatusBadge", () => {
  it.each(WORK_ORDER_STATUSES)(
    "%s shows a text label and an icon, not color alone",
    async (status) => {
      const screen = await render(<StatusBadge status={status} />);
      const badge = screen.getByText(STATUS_META[status].label);
      await expect.element(badge).toBeVisible();
      const el = badge.element();
      expect(el.querySelector("svg[aria-hidden]")).not.toBeNull();
      expect(el.getAttribute("data-status")).toBe(status);
    },
  );
});

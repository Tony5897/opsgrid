# Permission matrix

Generated from `internal/platform/authz`. Do not edit by hand — run `go generate ./internal/platform/authz/...`.

Owner is allow-all (`*`): every catalog permission. `platform.operate` is owner-only (Platform Health / DLQ; IMPLEMENTATION_PLAN §4.9).

| Permission | owner | admin | dispatcher | technician | viewer |
|---|---|---|---|---|---|
| `asset.create` | ✓ | ✓ |  |  |  |
| `asset.delete` | ✓ | ✓ |  |  |  |
| `asset.read` | ✓ | ✓ | ✓ |  | ✓ |
| `asset.read_assigned` | ✓ |  |  | ✓ |  |
| `asset.update` | ✓ | ✓ |  |  |  |
| `attachment.upload_assigned` | ✓ |  |  | ✓ |  |
| `audit.read` | ✓ | ✓ |  |  |  |
| `customer.create` | ✓ | ✓ |  |  |  |
| `customer.delete` | ✓ | ✓ |  |  |  |
| `customer.read` | ✓ | ✓ | ✓ |  | ✓ |
| `customer.read_assigned` | ✓ |  |  | ✓ |  |
| `customer.update` | ✓ | ✓ |  |  |  |
| `member.manage` | ✓ | ✓ |  |  |  |
| `member.read` | ✓ | ✓ |  |  |  |
| `organization.read` | ✓ | ✓ |  |  |  |
| `platform.operate` | ✓ |  |  |  |  |
| `report.read` | ✓ | ✓ | ✓ |  |  |
| `schedule.manage` | ✓ | ✓ | ✓ |  |  |
| `schedule.read` | ✓ | ✓ | ✓ |  | ✓ |
| `work_order.assign` | ✓ | ✓ | ✓ |  |  |
| `work_order.complete_assigned` | ✓ |  |  | ✓ |  |
| `work_order.create` | ✓ | ✓ | ✓ |  |  |
| `work_order.note_assigned` | ✓ |  |  | ✓ |  |
| `work_order.read` | ✓ | ✓ | ✓ |  | ✓ |
| `work_order.read_assigned` | ✓ |  |  | ✓ |  |
| `work_order.start_assigned` | ✓ |  |  | ✓ |  |
| `work_order.update` | ✓ | ✓ | ✓ |  |  |

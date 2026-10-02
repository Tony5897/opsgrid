## What & why

<!-- One paragraph: the problem and the approach. Link the gate / work package. -->

## Evidence checklist

- [ ] Tests at the right layer (unit / integration / proof / E2E) were added or updated
- [ ] `api/openapi.yaml` updated and `make generate` run (if the API changed)
- [ ] Migrations pass squawk and are expand/contract-safe (if the schema changed)
- [ ] `docs/evidence/ledger.md` updated (if a guarantee gained proof)
- [ ] ADR added or updated (if a decision was made)
- [ ] UI: Storybook stories for all states, light + dark screenshots, keyboard path verified
- [ ] No new lint suppressions without a justification comment

## Screenshots / traces / numbers

<!-- UI screenshots, trace links, benchmark deltas -->

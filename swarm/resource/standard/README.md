# Standard (Stable)

The Standard state represents production-ready, well-tested releases that are unlikely to change in patch terms.

## When to use
- Production-ready features with a stable interface.
- Customer-facing releases and major milestones.

## Lifecycle and rights
- Generally read-only in active environments; updates occur through formal release cycles.
- Promotions from Pilot or Incubator when features reach stability and repeatable outcomes.

## Promotion to Core
- Considered core means the platform is foundational and broadly supported.
- No major churn expected; minor updates follow standard release cadence.

## Editability and promotion
- Edits typically occur via the release governance process; direct edits to Standard are discouraged.
- Promote to Core when the feature set is fully stabilized and ready for long-term support.

## Quick example (config-like)
- State: standard
- label: Stable
- editable_by: team
- promotion: core
- description: "Production-ready, stable release."

# Incubator (Preview)

The Incubator state contains features or builds not fully finalized yet. It is useful for internal testing and controlled exposure to a broader audience.

## When to use
- Early access for internal QA, experimentation, or stakeholder previews.
- Features with churn or ongoing integration.

## Lifecycle and rights
- Editable by the team, with a focus on feedback-driven iteration.
- Promotions to Standard when stable; canary-style gradual rollouts for external users.

## Promotion to Standard
- Reaches stable behavior with repeatable outcomes across releases.
- Passes predefined acceptance criteria and QA checks.

## Editability and promotion
- Incubator holdings can be updated in place in ./swarm/resource/incubator.
- To promote, update metadata to the next state (e.g., next_state: standard) and document rationale.

## Quick example (config-like)
- State: incubator
- label: Incubator
- editable_by: team
- promotion: standard
- description: "Preview state for ongoing development and internal testing."

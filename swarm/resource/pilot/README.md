# Pilot (Ad hoc Release)

"Pilot" is an automatically created, one-off release snapshot intended for rapid testing and exploration. It is editable by anyone on the team and designed for short-lived use.

## When to use
- Quick experiments, demos, feature validations, and on-the-fly checks.
- Temporary visibility for stakeholders or internal demos.

## Lifecycle and rights
- Auto-generated and writable by all team members.
- Intended for single-use; if reused across releases or persisted beyond a short window, consider promotion to Incubator.

## Promotion to Incubator
- Reused in two or more distinct releases within 30 days.
- Involves inputs from multiple teams.
- Remains active for more than 14 days without a clear single-use conclusion.

## Editability and promotion
- Pilot snapshots live under ./swarm/resource/pilot and can be edited directly.
- Promote by updating metadata to the next state (e.g., next_state: incubator) and documenting the rationale.

## Quick example (config-like)
- State: ad_hoc
- label: Pilot
- auto_generated: true
- editable_by: all
- lifespan: one_time
- next_state: incubator
- description: "Automatically created, temporary release snapshot; editable by anyone; intended for single-use experiments."

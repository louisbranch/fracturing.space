---
title: "Worker Service Topology"
parent: "Project"
nav_order: 11
---

# Worker Service Topology

This document defines how background jobs should be split between
service-local workers and a dedicated worker service.

It also defines the prerequisite architecture for onboarding welcome
notifications.

## Purpose

Introduce a dedicated worker service with its own operational database for
robust retries, visibility, and replay of cross-service workflows, while
keeping domain-owned maintenance jobs inside the service that owns the data.

## Decision Summary

1. Add a dedicated `worker` service for cross-service orchestration jobs.
2. Keep service-local maintenance jobs local to each owning service.
3. Use producer-local outbox writes for durable handoff into worker processing.
4. Require idempotency keys and bounded retries for every worker handler.

## Job Placement Rules

### Keep local to owning service

Use service-local workers when the job:

- only touches that service's own database
- is tightly coupled to in-process caches/subscriptions
- does not require cross-service orchestration

Examples in current codebase:

- auth OAuth transient cleanup
- web cache invalidation and campaign subscription sync
- game projection apply outbox workers

### Move to dedicated worker service

Use dedicated worker service when the job:

- orchestrates across two or more services
- needs durable retries independent of request latency
- benefits from shared backoff/dead-letter/observability controls

Primary initial example:

- signup completed -> create onboarding welcome notification

## Worker Service Responsibilities

The dedicated worker service owns:

- job runtime state (leased jobs, attempts, next retry time)
- retry policy enforcement (exponential backoff + jitter)
- dead-letter and replay controls
- operator visibility (queue depth, oldest age, failure reasons)

The worker service does not own:

- user, auth, game, or notification domain authority
- direct writes to other services' databases

## Durable Handoff Contract

To preserve service boundaries, producers write to their own outbox in the same
transaction as the authoritative state change.

For onboarding:

1. Auth service records `signup_completed` and an auth outbox row atomically.
2. Worker ingests pending outbox items via auth API contract.
3. Worker calls notifications API `CreateNotificationIntent`.
4. Worker marks job success/failure in worker DB and ack progress to auth outbox
   contract.

This means dedicated workers reduce retry burden, but producers still carry a
small outbox responsibility for correctness.

## Retry, Idempotency, and Safety

Every worker handler must define:

- idempotency key format
- retry classing (transient vs permanent)
- max attempts
- dead-letter handling

Onboarding welcome defaults:

- topic: `auth.onboarding.welcome`
- source: `auth`
- dedupe key: `welcome:user:<user_id>:v1`
- retry: exponential backoff with jitter
- dead-letter after bounded attempts with operator replay path

## Onboarding Prerequisite

Treat worker platform readiness as a prerequisite for onboarding welcome
notification rollout.

Minimum gates:

1. Dedicated worker service exists with durable retry storage.
2. Auth service can durably emit `signup_completed` outbox entries.
3. Worker consumer for auth outbox is implemented with lease + retry.
4. Notifications call is idempotent using dedupe key.
5. Metrics and dead-letter replay are available before enablement.

Only after these gates should onboarding welcome notification be enabled in
production.

## Rollout Phases

### Phase 1: Platform foundation

- create worker service skeleton and DB schema
- implement generic lease/retry/dead-letter loop
- add metrics and admin inspection endpoints

### Phase 2: Auth producer outbox

- add auth outbox schema for integration events
- emit `signup_completed` at true signup completion boundary
- expose pull/ack contract for worker ingestion

### Phase 3: Onboarding handler

- implement worker handler that calls notifications service
- enforce dedupe key and retry policy
- canary rollout with dead-letter alerting

### Phase 4: Expansion

- add more cross-service workflows to worker service
- keep service-local maintenance jobs local unless they become orchestration jobs

## Non-Goals

- Centralizing all periodic tasks into one global service.
- Allowing worker service to bypass service APIs and write foreign DB records.
- Replacing existing domain event authority with worker-owned state.

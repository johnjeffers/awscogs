# Load Balancer Request Volume and Bandwidth Plan

## Objective

Add load balancer traffic visibility to awscogs so users can see:

- Request volume (or flow volume for NLB)
- Bandwidth/processed bytes
- Time-windowed usage (starting with `1h` and `24h`)

This feature should complement existing static ELB hourly cost reporting, not replace it.

## Current Baseline

The current app:

- Discovers ALB/NLB/CLB metadata and static hourly base pricing
- Returns ELB data through `/api/v1/costs/elb`
- Renders ELB table columns for account/region/name/type/scheme/state/hourly-daily-monthly cost
- Does not call CloudWatch today

## Scope (Phase 1)

In scope:

- Extend ELB payloads with usage metrics from CloudWatch
- Add `usageWindow` query support (`1h`, `24h`) on ELB cost endpoint
- Show usage metrics in ELB table
- Add IAM and docs updates
- Add tests for backend mapping + frontend rendering

Out of scope (later phases):

- Full time-series charting by minute/hour
- Variable ELB data-processing or LCU/NLCU cost estimation
- Cross-resource correlation (target groups, instances, pods)

## Metric Semantics by LB Type

Use CloudWatch metrics that are stable and easy to explain:

- ALB (`AWS/ApplicationELB`)
  - Volume metric: `RequestCount` (Sum)
  - Bandwidth metric: `ProcessedBytes` (Sum)
- NLB (`AWS/NetworkELB`)
  - Volume metric: `NewFlowCount` (Sum) (label as flows, not requests)
  - Bandwidth metric: `ProcessedBytes` (Sum)
- CLB (`AWS/ELB`)
  - Volume metric: `RequestCount` (Sum)
  - Bandwidth metric: `EstimatedProcessedBytes` (Sum)

UI labeling must reflect semantic differences:

- ALB/CLB: "Requests"
- NLB: "Flows"

## Data Model Changes

### Backend type updates

File: `backend/internal/types/types.go`

Add fields to `LoadBalancer`:

- `UsageWindow string` (e.g. `1h`, `24h`)
- `UsageStart string` (RFC3339)
- `UsageEnd string` (RFC3339)
- `RequestVolume float64` (ALB/CLB requests, NLB flows)
- `RequestMetricName string` (`RequestCount` or `NewFlowCount`)
- `BandwidthBytes float64`
- `BandwidthMetricName string` (`ProcessedBytes` or `EstimatedProcessedBytes`)
- `UsageStatus string` (`ok`, `partial`, `unavailable`)
- `UsageError string` (optional diagnostic message)

Notes:

- Keep existing fields untouched to avoid breaking existing consumers.
- Use floats for CloudWatch sums; format as integers in UI where appropriate.
- Define `UsageStatus` values as Go constants to avoid string drift (`ok`, `partial`, `unavailable`).

### Frontend type updates

File: `frontend/src/types/cost.ts`

Mirror the new ELB usage fields in the `LoadBalancer` interface.

## API Changes

### Endpoint behavior

Endpoint remains:

- `GET /api/v1/costs/elb`

Add optional query params:

- `usageWindow=1h|24h` (default `1h`)
- `includeUsage=true|false` (default `false`)

Rationale:

- Defaulting `includeUsage` to `false` avoids latency surprises for existing clients of `/costs/elb`.
- Frontend can explicitly opt in with `includeUsage=true`.
- Keeps future extension path open for global `/costs` usage expansion.

### Validation rules

- Invalid window returns `400` with clear message
- Missing CloudWatch permission returns `200` with `UsageStatus=unavailable` per LB (not hard fail entire response)
- Empty metric data returns zeros with `UsageStatus=partial` and a reason in `UsageError`

## Backend Implementation Plan

### 1) Add CloudWatch dependency

Files:

- `backend/go.mod`
- `backend/internal/aws/discovery.go`

Actions:

- Add AWS SDK CloudWatch service module.
- Import CloudWatch client in discovery package.

### 2) Define metric namespaces and dimensions explicitly

File:

- `backend/internal/aws/discovery.go`

Implement a single mapping function for metric metadata:

- ALB:
  - Namespace: `AWS/ApplicationELB`
  - Dimension: `LoadBalancer=app/<name>/<id>` (resource part of ARN)
  - Metrics: `RequestCount`, `ProcessedBytes`
- NLB:
  - Namespace: `AWS/NetworkELB`
  - Dimension: `LoadBalancer=net/<name>/<id>` (resource part of ARN)
  - Metrics: `NewFlowCount`, `ProcessedBytes`
- CLB:
  - Namespace: `AWS/ELB`
  - Dimension: `LoadBalancerName=<lb-name>`
  - Metrics: `RequestCount`, `EstimatedProcessedBytes`

### 3) Add usage retrieval and enrichment as a separate step

Files:

- `backend/internal/aws/discovery.go`
- `backend/internal/api/handlers/costs.go`

Actions:

- Keep `discoverELB` focused on metadata + static pricing only.
- Add a separate `enrichELBUsage(...)` step after resource discovery in `GetELBCosts`.
- This separates usage TTL from resource metadata TTL and avoids coupling to existing resource cache behavior.

### 4) Use `GetMetricData` with explicit rationale and batching

File:

- `backend/internal/aws/discovery.go`

Decision:

- Use `GetMetricData` (not `GetMetricStatistics`) because it supports multi-metric batching.
- `GetMetricStatistics` remains simpler for tiny environments, but does not batch multiple metrics/LBs in one request.

Implementation:

1. Group load balancers by account+region and query in batches.
2. Build `MetricDataQuery` entries for volume and bandwidth metrics.
3. Use stable periods:
   - `1h` window -> `300s`
   - `24h` window -> `3600s`
4. Aggregate `Sum` values over returned datapoints.
5. Map results back to each LB and set usage status fields.

### 5) Add dedicated usage cache (account+region+window)

File:

- `backend/internal/aws/discovery.go`

Add cache entry keyed by:

- `accountID|region|window`

Stored value:

- `map[lbIdentifier]ELBUsageData`

TTL recommendation:

- `1h` window cache TTL: 2-5 minutes
- `24h` window cache TTL: 10-15 minutes

Reason:

- Matches existing account+region cache style more closely than per-LB cache.
- Reduces CloudWatch calls across repeated UI refreshes.

### 6) Add mandatory concurrency control

File:

- `backend/internal/aws/discovery.go`

Actions:

- Add a semaphore for CloudWatch usage fetches.
- Initial limit: `10` concurrent CloudWatch requests (configurable later).
- Keep existing discovery goroutines, but gate usage queries through the semaphore.

### 7) Extend handler query parsing

File:

- `backend/internal/api/handlers/costs.go`

Actions:

- Parse `usageWindow` and `includeUsage`.
- Validate `usageWindow` values (`1h`, `24h`) and return `400` on invalid values.
- Only execute usage enrichment when `includeUsage=true`.
- Keep backward-compatible defaults with usage disabled unless requested.

## Frontend Implementation Plan

### 1) API client support

File:

- `frontend/src/services/api.ts`

Actions:

- Extend `getELBCosts` signature to include usage options
- Append query params when provided

### 2) Table columns and formatting

File:

- `frontend/src/components/costs/CostTable.tsx`

Add ELB columns:

- Requests/Flows (window-scoped)
- Bandwidth (bytes, rendered as KiB/MiB/GiB/TiB)
- Usage status badge when unavailable/partial

Sorting:

- Add sort keys for request volume and bandwidth

### 3) Window selector in dashboard

File:

- `frontend/src/components/costs/CostDashboard.tsx`

Actions:

- Add control for `1h` / `24h` when ELB tab active (or globally if simpler)
- Trigger ELB refresh on window change

## IAM / Deployment Changes

### Required AWS permissions

Execution role (or assumed role) needs:

- `cloudwatch:GetMetricData`

Optional:

- `cloudwatch:ListMetrics` (not required if dimensions are known)

### Documentation updates

Files:

- `README.md`
- `helm/values.yaml` comments or helm docs

Document:

- New permission requirements
- Metric semantics differences between ALB/CLB and NLB
- Meaning of usage window and freshness

## Performance and Reliability

### Expected load

Without batching/caching, usage calls can scale quickly with account x region x LB count. Mitigations:

- Batch metric queries per account+region request using `GetMetricData`
- Use cache with short TTL
- Enforce concurrency limit (semaphore, initial value `10`)

### Failure behavior

- Never fail full ELB endpoint because one usage query fails
- Return best-effort usage per LB with explicit status/error fields
- Log warnings with account/region/lb identifiers for troubleshooting

## Testing Plan

### Backend tests

Areas:

- Metric mapping per LB type (namespace, metric names, dimensions)
- Window parsing and validation
- Aggregation logic with sparse/missing datapoints
- Error-to-status mapping (`ok`, `partial`, `unavailable`)
- Throttling handling (`Throttling` -> `UsageStatus=unavailable`, endpoint still returns `200`)

### Frontend tests

Areas:

- ELB table renders request/flow labels correctly by LB type
- Byte formatter correctness and sort behavior
- Usage status badge rendering
- Window selector state and API call propagation

### Manual validation checklist

1. Run with at least one ALB and one NLB in selected accounts.
2. Verify ALB shows requests + processed bytes.
3. Verify NLB shows flows + processed bytes.
4. Remove CloudWatch permission and confirm graceful degradation.
5. Confirm no regression in existing ELB hourly cost totals.

## Rollout Plan

1. Merge backend model + CloudWatch retrieval behind `includeUsage` toggle (`default=false`).
2. Update frontend to request `includeUsage=true` with default `usageWindow=1h`.
3. Monitor logs for throttling/errors in staging.
4. Tune cache TTL and concurrency if needed.
5. Document known limitations.

## Risks and Mitigations

- Risk: Metric confusion for NLB request semantics
  - Mitigation: Explicitly label as flows and document metric source.
- Risk: CloudWatch throttling in large orgs
  - Mitigation: Add cache + concurrency limits + minimal default window.
- Risk: Missing data for newly created/inactive LBs
  - Mitigation: Return `partial/unavailable` status instead of hard errors.
- Risk: Response payload growth
  - Mitigation: Keep usage fields lightweight and optional via `includeUsage`.
- Risk: CloudWatch API retrieval cost growth (especially with `GetMetricData`)
  - Mitigation: Batch per account+region, cache aggressively, and keep default `includeUsage=false`.

## Acceptance Criteria

- `/api/v1/costs/elb?usageWindow=1h&includeUsage=true` returns usage fields for each LB.
- ELB table displays request/flow volume and bandwidth for selected window.
- Existing ELB cost columns remain accurate and unchanged.
- Feature works across ALB, NLB, and CLB with clear labels.
- Missing CloudWatch permissions do not break the endpoint; status is explicit.

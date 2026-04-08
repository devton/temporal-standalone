# Temporal Production Checklist

## Pre-Launch

### Workflow Code Review
- [ ] All code follows determinism rules (no time.Now, rand, global state)
- [ ] Activities are idempotent (handle duplicate execution)
- [ ] Proper timeout values for all activities
- [ ] Retry policies configured with appropriate backoff
- [ ] Long activities implement heartbeating
- [ ] Signal handlers registered before use
- [ ] No blocking operations in workflow code

### Configuration
- [ ] Workflow execution timeout set appropriately
- [ ] Run timeout prevents runaway workflows
- [ ] Task timeout matches activity duration
- [ ] Heartbeat timeout < activity duration / 10

### Error Handling
- [ ] Business errors use ApplicationError
- [ ] Retryable vs non-retryable errors distinguished
- [ ] Compensation logic for saga patterns
- [ ] Cancellation handlers implemented

## Security

### Authentication & Authorization
- [ ] mTLS enabled for server-worker communication
- [ ] JWT/OIDC integration for UI access (Temporal Cloud)
- [ ] Namespace-level isolation
- [ ] Authorizer plugin configured (self-hosted)

### Data Protection
- [ ] Sensitive data encrypted at rest (Codec Server)
- [ ] Encryption keys managed securely (Vault, AWS KMS)
- [ ] PII not in workflow IDs or search attributes
- [ ] Audit logging enabled

### Network
- [ ] Firewall rules restrict Temporal port access
- [ ] Workers in private subnets
- [ ] UI behind auth proxy (self-hosted)

## Operations

### Monitoring & Alerting
- [ ] Prometheus metrics exported
- [ ] Grafana dashboards configured
- [ ] Alert rules for:
  - [ ] High workflow failure rate
  - [ ] Worker lag (task queue backlog)
  - [ ] Persistence latency
  - [ ] History size approaching limits

### Logging
- [ ] Structured logging configured
- [ ] Log aggregation (ELK, Splunk, CloudWatch)
- [ ] Sensitive data filtered
- [ ] Correlation IDs for tracing

### Backup & Recovery
- [ ] Database backups configured
- [ ] Backup restore tested
- [ ] Retention period set appropriately
- [ ] Archival configured (if needed)

## Performance

### Scaling
- [ ] History shards adequate for workload
- [ ] Workers autoscale based on load
- [ ] Task queue partitioning strategy
- [ ] Continue-as-new for long workflows

### Optimization
- [ ] Batch activities for high throughput
- [ ] Local activities for fast operations
- [ ] Search attributes indexed for queries
- [ ] Visibility store tuned

## Reliability

### High Availability
- [ ] Multi-zone deployment (self-hosted)
- [ ] Replication configured (Temporal Cloud)
- [ ] Failover tested
- [ ] Worker versioning strategy

### Disaster Recovery
- [ ] RTO/RPO defined
- [ ] Multi-region replication (Temporal Cloud)
- [ ] DR runbook documented
- [ ] Regular DR drills

## Maintenance

### Deployment
- [ ] Blue/green worker deployment
- [ ] Backward-compatible workflow changes
- [ ] Feature flags for new workflows
- [ ] Rollback procedure documented

### Observability
- [ ] Workflow replay tested
- [ ] Reset capability verified
- [ ] History export configured
- [ ] Capacity planning reviewed

---

## Key Metrics to Monitor

| Metric | Warning | Critical |
|--------|---------|----------|
| Workflow Failure Rate | > 1% | > 5% |
| Task Queue Backlog | > 1000 | > 10000 |
| History Size | > 40K events | > 50K events |
| Persistence Latency | > 100ms | > 500ms |
| Worker Heartbeat Miss | > 1% | > 5% |

---

## Common Pitfalls

1. **Non-deterministic code** - Causes replay failures
2. **Large history** - Use continue-as-new
3. **Missing heartbeats** - Activity appears hung
4. **Wrong timeout values** - Either too short or too long
5. **No idempotency** - Duplicate side effects
6. **Unbounded signals** - Memory leak
7. **Blocking calls** - Worker stalls

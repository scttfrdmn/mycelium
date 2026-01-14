# Spawn Development Roadmap

## Current Status

Spawn has a solid foundation for single-instance workflows with excellent developer experience. The core features (launch, connect, TTL, hibernation, DNS, AMI management) are production-ready. However, several critical features are missing for team/production use and modern cloud workloads.

**Completion Status:**
- **Single Instance Lifecycle:** ✅ 100% - Launch, connect, terminate, extend, hibernate
- **Cost Management:** ✅ 90% - TTL, idle detection, hibernation
- **DNS Management:** ✅ 100% - spore.host subdomains, auto-registration
- **AMI Management:** ✅ 100% - Create, list, health checks
- **Multi-Instance Coordination:** ❌ 0% - No job arrays
- **Cost Optimization:** ❌ 0% - No spot instances
- **Security:** ❌ 0% - No IAM instance profiles
- **Team Features:** ⚠️ 30% - Dashboard foundation exists but incomplete

## Immediate Priorities (Next 4-8 Weeks)

These three features are critical blockers for production adoption:

### 1. Job Arrays 🎯 **HIGHEST PRIORITY**

**Status:** Design complete, ready for implementation

**Plan:** [Job Arrays Plan](/.claude/plans/jolly-dreaming-neumann.md)

**Why Critical:**
- Blocks distributed ML training workloads
- Prevents batch processing use cases
- No parallel computing support
- Competitors have this (AWS Batch, Slurm, Kubernetes)

**Impact:**
- Enables 90% of modern cloud computing workloads
- Unlocks team collaboration features
- Foundation for auto-scaling and cluster management

**Estimated Effort:** 2-3 weeks
- Phase 1 (Core Launch): 1 week
- Phase 2 (Peer Discovery): 3-4 days
- Phase 3 (Group DNS): 2-3 days
- Phase 4 (Management): 2-3 days

**Dependencies:** None - builds on existing infrastructure

**Key Features:**
- Launch N instances with single command
- Automatic peer discovery
- Group DNS (one name for all instances)
- MPI-style coordination (rank, size, peers)
- Group management (terminate/extend entire array)

**Use Cases Unlocked:**
- Distributed ML training (PyTorch DDP, Horovod)
- Parallel data processing
- Batch job workloads
- Cluster computing
- Multi-node simulations

---

### 2. Spot Instance Support 💰 **HIGH PRIORITY**

**Status:** Design complete, ready for implementation

**Plan:** [SPOT_INSTANCES.md](SPOT_INSTANCES.md)

**Why Critical:**
- 70-90% cost savings for fault-tolerant workloads
- Competitive necessity (all cloud tools support spot)
- Makes spawn cost-effective for large-scale use
- Critical for budget-conscious teams

**Impact:**
- Massive cost reduction for ML training, batch jobs
- Enables longer-running workloads within budget
- Spot + job arrays = powerful combo for distributed work

**Estimated Effort:** 1-2 weeks
- Launch with spot: 2-3 days
- Interruption monitoring: 2-3 days
- Checkpoint execution: 2 days
- Testing and integration: 2-3 days

**Dependencies:**
- Works standalone
- Enhanced by job arrays (spot arrays)

**Key Features:**
- `--spot` flag for instant savings
- Interruption handling (2-minute warning)
- Checkpoint script execution
- Fallback to on-demand
- Mixed spot/on-demand job arrays

**Use Cases Unlocked:**
- Cost-effective ML training
- Batch processing at scale
- CI/CD on budget
- Development environments (massive savings)

---

### 3. IAM Instance Profiles 🔐 **HIGH PRIORITY**

**Status:** Design complete, ready for implementation

**Plan:** [IAM_INSTANCE_PROFILES.md](IAM_INSTANCE_PROFILES.md)

**Why Critical:**
- No secure way to access AWS services from instances
- Users embedding credentials (security risk)
- Blocks production workloads requiring S3, DynamoDB, etc.
- Audit compliance requires proper IAM

**Impact:**
- Enables secure cloud-native patterns
- Unlocks S3 data pipelines
- Enables CloudWatch logging/metrics
- Proper security posture for production

**Estimated Effort:** 1-2 weeks
- IAM role creation: 3-4 days
- Policy templates: 2 days
- Role reuse logic: 2 days
- Testing and docs: 2-3 days

**Dependencies:** None - standalone feature

**Key Features:**
- Simple `--iam-policy s3:ReadOnly` syntax
- Automatic role creation and reuse
- Built-in policy templates for common services
- Custom policy file support
- No credentials in code

**Use Cases Unlocked:**
- S3 data pipelines (read datasets, write results)
- CloudWatch monitoring
- DynamoDB applications
- ECR container workflows
- Secrets Manager integration

---

## Implementation Strategy

### Recommended Order

**Week 1-3: Job Arrays**
- Week 1: Phase 1 (Core Launch) + Phase 2 (Peer Discovery)
- Week 2: Phase 3 (Group DNS) + Phase 4 (Management)
- Week 3: Testing, documentation, edge cases

**Week 4-5: IAM Instance Profiles**
- Week 4: Core IAM role creation, policy templates
- Week 5: Testing, integration with job arrays

**Week 6-7: Spot Instances**
- Week 6: Launch with spot, interruption monitoring
- Week 7: Checkpoint execution, spot + job arrays integration

**Week 8: Integration & Polish**
- Test all three features together
- Spot + IAM + job arrays use cases
- Documentation updates
- Example workflows

### Why This Order?

1. **Job Arrays First:**
   - Highest impact - unlocks most use cases
   - Foundation for other features
   - Spot arrays and IAM arrays both depend on job arrays
   - Longest development time

2. **IAM Second:**
   - Independent of spot
   - Can test with job arrays immediately
   - Simpler than spot (no interruption handling)
   - High security value

3. **Spot Last:**
   - Can leverage job arrays (already implemented)
   - Can leverage IAM (for checkpoint scripts in S3)
   - Requires most testing (interruption scenarios)
   - Builds on top of both other features

### Integration Benefits

**Job Arrays + Spot:**
```bash
# 8-instance spot GPU training cluster
spawn launch --count 8 --job-array-name training \
  --instance-type g5.xlarge --spot \
  --on-interruption checkpoint \
  --checkpoint-script s3://bucket/save.sh
```

**Job Arrays + IAM:**
```bash
# Distributed data processing with S3 access
spawn launch --count 16 --job-array-name processing \
  --iam-policy s3:ReadWrite,dynamodb:WriteOnly \
  --command "python process.py --rank \$JOB_ARRAY_INDEX"
```

**All Three Together:**
```bash
# Cost-effective, secure, distributed training
spawn launch --count 32 --job-array-name llm-training \
  --instance-type g5.xlarge --spot \
  --iam-policy s3:FullAccess,logs:WriteOnly \
  --on-interruption checkpoint \
  --checkpoint-script s3://bucket/checkpoint.sh \
  --user-data @train.sh
```

---

## Medium-Term Priorities (2-4 Months)

### 4. Web Dashboard
**Status:** Foundation exists, incomplete

**Dependencies:** Job arrays (to display arrays in UI)

**Effort:** 3-4 weeks

**Features:**
- React frontend for instance management
- Visual job array status
- Mobile-friendly design
- Team collaboration features

### 5. Volume Management
**Status:** Not started

**Effort:** 2-3 weeks

**Features:**
- Attach/detach EBS volumes
- Volume snapshots
- Persistent storage for instances
- Volume tagging and discovery

### 6. Cost Tracking
**Status:** Not started

**Effort:** 2 weeks

**Features:**
- Cost estimation on launch
- Monthly spending reports
- Budget alerts
- Cost breakdown by tag/project

### 7. Network Configuration
**Status:** Basic security groups only

**Effort:** 1-2 weeks

**Features:**
- VPC/subnet selection
- Security group customization
- Network ACL configuration
- Elastic IP management

---

## Long-Term Vision (4+ Months)

### Auto-Scaling Job Arrays
Automatically maintain N running instances, replacing spot interruptions or failures.

### Multi-Region Job Arrays
Coordinate instances across regions for global workloads.

### Template System
Save and reuse complex launch configurations.

### Marketplace
Pre-built AMIs for popular frameworks (PyTorch, TensorFlow, Ray, etc.).

### Integration Ecosystem
- Terraform provider
- GitHub Actions
- Kubernetes operator
- VS Code extension

---

## Success Metrics

### Phase 1 Success (After Job Arrays, IAM, Spot)
- [ ] Can launch 100-instance job array in <2 minutes
- [ ] Spot instances working with 2-minute warning handling
- [ ] IAM roles created and attached automatically
- [ ] All three features work together
- [ ] 5+ documented use cases
- [ ] 80%+ test coverage

### Production Readiness
- [ ] 1000+ instances launched successfully
- [ ] <1% failure rate on launches
- [ ] Cost savings averaging 60%+ with spot
- [ ] Zero credential leaks (all via IAM)
- [ ] Dashboard shows real-time status
- [ ] Multi-team deployment in production

### Market Validation
- [ ] 10+ external users/teams
- [ ] Positive feedback on UX
- [ ] Feature parity with AWS Batch (for core use cases)
- [ ] Differentiation via simplicity
- [ ] Community contributions

---

## Risk Mitigation

### Job Arrays Risks
**Risk:** Partial launch failures leave orphaned instances
**Mitigation:** Automatic cleanup on failure, detailed error messages

**Risk:** DNS propagation delays
**Mitigation:** Poll for DNS availability, provide fallback IP access

**Risk:** Large arrays (100+) overwhelm EC2 API
**Mitigation:** Rate limiting, batch operations, progress indicators

### Spot Instance Risks
**Risk:** Interruption during critical operation
**Mitigation:** User controls via checkpoint scripts, clear documentation

**Risk:** Spot unavailable in region
**Mitigation:** Fallback to on-demand, multi-instance-type selection

**Risk:** Cost savings less than expected
**Mitigation:** Show savings estimate before launch, cost tracking

### IAM Risks
**Risk:** Overly permissive policies
**Mitigation:** Least-privilege templates, warnings on broad access

**Risk:** Role name conflicts
**Mitigation:** Hash-based naming, reuse existing roles

**Risk:** IAM eventual consistency delays
**Mitigation:** Retry logic, wait for role propagation

---

## Testing Strategy

### Unit Tests
- All public functions
- Error paths
- Edge cases
- 80%+ coverage target

### Integration Tests
- End-to-end workflows
- AWS API mocks
- Partial failure scenarios
- Cross-feature integration

### Load Tests
- 100-instance job arrays
- Concurrent launches
- API rate limit handling

### Production Tests
- Canary deployments
- Gradual rollout
- Monitoring and alerts

---

## Documentation Plan

### User Guides
- Getting started with job arrays
- Spot instances best practices
- IAM security patterns
- Cost optimization guide
- Troubleshooting common issues

### API Documentation
- Command reference
- Flag descriptions
- Exit codes
- Error messages

### Architecture Docs
- System design
- Component interactions
- DNS architecture
- Tag conventions

### Examples
- ML training clusters
- Batch processing pipelines
- CI/CD workflows
- Data processing

---

## Resource Requirements

### Development
- 1 senior engineer (full-time, 8 weeks)
- OR 2 engineers (part-time, 8 weeks)

### Testing
- AWS test account
- Budget for EC2 testing: $500-1000/month
- Spot instance testing across regions

### Infrastructure
- CI/CD pipeline updates
- Integration test infrastructure
- Documentation hosting

---

## Summary

The three immediate priority features (job arrays, spot instances, IAM profiles) are **critical blockers** for production adoption. Together they unlock:

✅ **Modern Workloads:** Distributed computing, ML training, batch processing
✅ **Cost Efficiency:** 70-90% savings with spot
✅ **Security:** Proper IAM patterns, no embedded credentials
✅ **Scale:** Hundreds of coordinated instances
✅ **Team Use:** Foundation for dashboard and collaboration

**Estimated Timeline:** 6-8 weeks for all three features

**Impact:** Transforms spawn from "convenient single-instance tool" to "production-ready cloud orchestration platform"

After these features, spawn will be competitive with AWS Batch, Slurm, and other batch computing tools while maintaining its signature simplicity and opinionated design.

# AWS EC2 Instance Types - December 2025 Reference

This document lists the latest AWS EC2 instance types available as of December 2025.

## Latest Generation Instances (2024-2025)

### General Purpose (M family)

#### 8th Generation - Graviton4 (2025)
- **m8g.medium** through **m8g.metal**
- Processor: AWS Graviton4 (ARM)
- Features: Best price-performance, DDR5 memory
- Use case: General workloads, web servers, app servers

#### 7th Generation - Intel (2024)
- **m7i.large** through **m7i.metal-48xl**
- Processor: Intel Sapphire Rapids (4th Gen Xeon)
- Features: Up to 192 vCPUs, DDR5 memory
- Use case: General compute, databases, caching

#### 7th Generation - AMD (2023-2024)
- **m7a.medium** through **m7a.metal-48xl**
- Processor: AMD EPYC Genoa (4th Gen)
- Features: Up to 192 vCPUs, competitive pricing
- Use case: General workloads, cost optimization

#### 7th Generation - Graviton3 (2022-2023)
- **m7g.medium** through **m7g.metal**
- Processor: AWS Graviton3 (ARM)
- Features: Up to 64 vCPUs, energy efficient
- Use case: General ARM workloads

### Compute Optimized (C family)

#### 8th Generation - Graviton4 (2025)
- **c8g.medium** through **c8g.metal**
- Processor: AWS Graviton4
- Features: Highest compute performance per watt
- Use case: CPU-intensive workloads, HPC

#### 7th Generation - Intel (2024)
- **c7i.large** through **c7i.metal-48xl**
- Processor: Intel Sapphire Rapids
- Features: Up to 192 vCPUs, high frequency
- Use case: Batch processing, gaming, scientific modeling

#### 7th Generation - AMD (2023-2024)
- **c7a.medium** through **c7a.metal-48xl**
- Processor: AMD EPYC Genoa
- Features: Up to 192 vCPUs, cost-effective
- Use case: High-performance computing, video encoding

#### 7th Generation - Graviton3 Network (2023)
- **c7gn.medium** through **c7gn.16xlarge**
- Processor: AWS Graviton3
- Features: 200 Gbps network bandwidth
- Use case: Network-intensive applications, 5G

### Memory Optimized (R family)

#### 8th Generation - Graviton4 (2025)
- **r8g.medium** through **r8g.metal**
- Processor: AWS Graviton4
- Features: DDR5, memory-intensive workloads
- Use case: In-memory databases, SAP HANA

#### 7th Generation - Intel (2024)
- **r7i.large** through **r7i.metal-48xl**
- Processor: Intel Sapphire Rapids
- Features: Up to 1536 GiB RAM
- Use case: Large databases, real-time analytics

#### 7th Generation - AMD (2023-2024)
- **r7a.medium** through **r7a.metal-48xl**
- Processor: AMD EPYC Genoa
- Features: Up to 1536 GiB RAM, cost-effective
- Use case: Memory-intensive applications

#### 7th Generation - Graviton3 (2022-2023)
- **r7g.medium** through **r7g.metal**
- Processor: AWS Graviton3
- Features: Up to 512 GiB RAM
- Use case: Memory-optimized ARM workloads

### Storage Optimized (I family)

#### 4th Generation - Intel (2023-2024)
- **i4i.large** through **i4i.metal**
- Processor: Intel Ice Lake
- Features: Local NVMe SSD, up to 30 TB storage
- Use case: NoSQL databases, data warehousing

#### 4th Generation - Graviton (2023)
- **im4gn.large** through **im4gn.16xlarge**
- Processor: AWS Graviton2
- Features: Local NVMe SSD, network optimized
- Use case: Storage-intensive ARM workloads

- **is4gen.medium** through **is4gen.8xlarge**
- Processor: AWS Graviton2
- Features: Local SSD
- Use case: Distributed file systems

### GPU Instances (P, G, Inf, Trn families)

#### P5 - Latest NVIDIA GPU (2024)
- **p5.48xlarge**
- GPU: 8x NVIDIA H100 (80GB each)
- Features: 640 GB GPU memory, 3200 Gbps networking
- Use case: Large language models, generative AI

#### G6 - NVIDIA L4/L40S (2024)
- **g6.xlarge** through **g6.48xlarge**
- GPU: NVIDIA L4 or L40S
- Features: Cost-effective graphics, AI inference
- Use case: Graphics workstations, video processing

#### Inf2 - AWS Inferentia2 (2023-2024)
- **inf2.xlarge** through **inf2.48xlarge**
- Accelerator: AWS Inferentia2 chips
- Features: High-throughput inference, low latency
- Use case: Large-scale ML inference, LLM deployment

#### Trn1 - AWS Trainium (2023-2024)
- **trn1.2xlarge** through **trn1.32xlarge**
- Accelerator: AWS Trainium chips
- Features: Optimized for ML training
- Use case: Deep learning model training

- **trn1n.32xlarge**
- Accelerator: AWS Trainium with enhanced networking
- Features: 1600 Gbps network bandwidth
- Use case: Distributed ML training

### High Performance Computing (HPC family)

#### HPC7g - Graviton3 (2023)
- **hpc7g.4xlarge**, **hpc7g.8xlarge**, **hpc7g.16xlarge**
- Processor: AWS Graviton3E
- Features: EFA networking, HPC optimized
- Use case: Weather modeling, CFD simulations

#### HPC7a - AMD (2024)
- **hpc7a.12xlarge**, **hpc7a.24xlarge**, **hpc7a.48xlarge**, **hpc7a.96xlarge**
- Processor: AMD EPYC Genoa
- Features: 4th Gen AMD, HPC workloads
- Use case: Large-scale simulations, research

## Popular Instance Types by Use Case

### Web Servers & Applications
- **t3a.medium** - Burstable, cost-effective
- **m7i.large** - Latest Intel, general purpose
- **m8g.large** - Latest Graviton4, best price-performance

### Databases
- **r7i.xlarge** - Memory optimized, Intel
- **r8g.xlarge** - Memory optimized, Graviton4
- **i4i.xlarge** - Local NVMe for high IOPS

### Microservices
- **t3a.micro** through **t3a.small** - Small workloads
- **m8g.medium** - Container workloads
- **c7a.large** - Compute-intensive services

### ML Inference
- **inf2.xlarge** - AWS Inferentia2
- **g6.2xlarge** - NVIDIA L4 GPU
- **c7i.4xlarge** - CPU inference

### ML Training
- **p5.48xlarge** - Largest GPU instance
- **trn1.32xlarge** - AWS Trainium
- **p4d.24xlarge** - Previous gen NVIDIA A100

### Big Data & Analytics
- **r7a.8xlarge** - Memory optimized, AMD
- **i4i.4xlarge** - Local storage for Hadoop/Spark
- **c7i.8xlarge** - Compute for data processing

## Architecture Comparison

### x86_64 (Intel/AMD)
- **Pros**: Wide software compatibility, mature ecosystem
- **Cons**: Higher cost, more power consumption
- **Best for**: Legacy applications, Windows workloads

### arm64 (Graviton)
- **Pros**: Better price-performance, energy efficient
- **Cons**: Some software may need recompilation
- **Best for**: Cloud-native apps, containers, open-source

## Recommended Default Instances (Dec 2025)

| Use Case | Budget | Balanced | Performance |
|----------|--------|----------|-------------|
| Web App | t3a.medium | m7i.large | m8g.xlarge |
| Database | r7a.large | r7i.xlarge | r8g.2xlarge |
| Compute | c7a.large | c7i.xlarge | c8g.2xlarge |
| ML Inference | inf2.xlarge | g6.2xlarge | inf2.8xlarge |
| ML Training | trn1.2xlarge | trn1.32xlarge | p5.48xlarge |

## Regional Availability Notes

### Generally Available Everywhere
- m7i, c7i, r7i families
- m8g, c8g, r8g families (most regions)
- t3, t3a families

### Limited Availability
- **p5.48xlarge**: us-east-1, us-west-2, eu-west-1 only
- **trn1n**: Select regions with enhanced networking
- **hpc7a**: HPC-focused regions

### Coming Soon (Q1 2025)
- m9g family (Graviton5 rumored)
- Enhanced inf3 instances
- Next-gen GPU instances

## Cost Optimization Tips

1. **Use Graviton4** (m8g, c8g, r8g) for up to 40% better price-performance
2. **Consider AMD** (m7a, c7a, r7a) for 10-15% cost savings vs Intel
3. **Right-size** using Truffle to find available alternatives
4. **Spot Instances** work great with newer generation types
5. **Savings Plans** provide significant discounts on 7th/8th gen

## Using These with Truffle

```bash
# Find latest Graviton4 instances
truffle search "m8g.*"
truffle search "c8g.*"
truffle search "r8g.*"

# Compare Intel vs AMD vs Graviton
truffle search "m7i.xlarge"
truffle search "m7a.xlarge"
truffle search "m7g.xlarge"
truffle search "m8g.xlarge"

# Find ML instances
truffle search "inf2.*"
truffle search "trn1.*"
truffle search "p5.*"

# Search by specs (find suitable alternatives)
truffle search "*" --min-vcpu 8 --min-memory 32 --architecture arm64
```

## References

- AWS Instance Types: https://aws.amazon.com/ec2/instance-types/
- Graviton: https://aws.amazon.com/ec2/graviton/
- ML Accelerators: https://aws.amazon.com/machine-learning/accelerators/

---

Last Updated: December 2025

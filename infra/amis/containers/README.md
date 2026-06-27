# spore.host container catalog (#290)

App images for the **container-based** app catalog. Each app is a Docker image
that runs on the shared `spore-dcv-base` AMI; there is **no per-app AMI**. This
replaces the old `infra/amis/<app>.pkr.hcl` per-app, per-region AMI builds whose
IDs all drifted into dangling/unshared state (#389).

## Why containers

| | Old (AMI per app per region) | Container catalog |
|---|---|---|
| Add an app | ~20 min Packer build × 9 regions | write a Dockerfile, `build-push.sh` once |
| Update a version | rebuild + reshare every region | `docker push` a new tag |
| Drift risk | 9 × N AMI IDs to keep shared (#389) | 1 base AMI per region, shared once |
| Multi-version | full rebuild | image tags (`paraview:5.13.2`, `5.12.1`) |

## Layout

```
containers/
├── build-push.sh        # build <app>:<version> and push to public ECR
├── paraview/
│   ├── Dockerfile        # ParaView + metacity + fullscreen entrypoint
│   └── entrypoint.sh      # the container CMD = the DCV session
└── <app>/ ...
```

## How a launch works

`spawn app launch paraview` →
1. picks `spore-dcv-base` AMI for the region (`base_amis` in `libs/catalog/catalog.yaml`),
2. user-data pre-pulls `public.ecr.aws/spore-host/paraview:<tag>`,
3. creates the DCV session with this as `--init`:
   ```sh
   docker run --rm --gpus all --network host -e DISPLAY=:0 \
     -v /tmp/.X11-unix:/tmp/.X11-unix public.ecr.aws/spore-host/paraview:<tag>
   ```
The base AMI provides DCV, the NVIDIA driver + X server on `:0`, Docker, and the
NVIDIA Container Toolkit (built by `../dcv-gpu-al2023.pkr.hcl`). The container
provides the app, its window manager, and the fullscreen launcher, drawing into
the host's DCV display via the bind-mounted X socket.

## Build & publish a new app/version

The app images are **x86_64** (the base AMI and app binaries are x86_64), so the
build must target `linux/amd64`. `build-push.sh` forces `--platform linux/amd64`
via buildx — on an arm64 host (Apple Silicon) that cross-builds under QEMU, which
works but is slow.

**Preferred: build natively on a throwaway x86 EC2 instance (via spawn).** No
emulation, native speed, and an instance role avoids local credential expiry.
Sketch (run in the ECR/infra account 812107987990):

```sh
spawn launch --instance-type c7i.2xlarge --region us-east-1 \
  --ttl 1h --terminate-on-complete \
  --command 'set -e; dnf install -y docker git && systemctl start docker &&
    git clone https://github.com/spore-host/spore-host &&
    cd spore-host/infra/amis/containers &&
    ./build-push.sh paraview 5.13.2'
# the instance has an instance role with ecr-public:* push perms.
```

**Local fallback** (no AWS / for a quick check; cross-builds on arm64):

```sh
./build-push.sh paraview 5.13.2            # build + push from your workstation
SPORE_BUILD_DRYRUN=true ./build-push.sh paraview 5.13.2   # build only, no push
```

Either way: then add `image`/`tag_default`/`tags_available` to
`libs/catalog/catalog.yaml` and cut a libs release.

## Base AMI (one-time per region)

```sh
# Build the base (GPU) AMI:
cd .. && packer build -var region=us-east-1 dcv-gpu-al2023.pkr.hcl
# Share it to the launch account — THE #389 FIX:
./share-base-ami.sh <ami-id> us-east-1 <launch-account-id>
```

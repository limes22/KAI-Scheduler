# KAI + HAMi-core (fractional GPU isolation)

This fork wires **HAMi-core (libvgpu)** into KAI so that fractional GPU pods get real
per-container caps on **GPU memory** (`CUDA_DEVICE_MEMORY_LIMIT`) and **SM/compute**
(`CUDA_DEVICE_SM_LIMIT`). No separate resource-isolator webhook is used and the NVIDIA
device-plugin is left untouched.

## How it works

1. **admission** (`pkg/admission/webhook/v1alpha2/hamicore`) — for every fractional GPU
   pod, injects the `kai-hami-vgpu` hostPath volume + `ld.so.preload` mount and the
   `CUDA_DEVICE_MEMORY_LIMIT` / `CUDA_DEVICE_SM_LIMIT` env vars (sourced from the
   capabilities ConfigMap).
2. **binder** (`pkg/binder/plugins/hamicore`) — at bind time, computes the caps from the
   bound GPU portion and node VRAM. SM limit accepts either `gpu-sm-percentage` (1-100)
   or `gpu-sm-cores` (÷ `nvidia.com/gpu.cores` → %). Memory cap is derived from the
   `gpu-fraction` / `gpu-memory` portion.
3. **kai-hami-libsync** DaemonSet — copies `libvgpu.so` + `ld.so.preload` onto every GPU
   node at `/usr/local/vgpu`, so the LD_PRELOAD'd hook actually enforces the caps.

## Install (Helm)

The chart deploys the libsync DaemonSet and enables the `hamicore` binder plugin
automatically when `hamiCore.enabled=true` (the default in this fork). `global.gpuSharing`
must be `true` (also the default here).

```bash
helm upgrade --install kai-scheduler ./deployments/kai-scheduler \
  --namespace kai-scheduler --create-namespace \
  --set global.registry=howdi2000 \
  --set global.tag=hami-sm \
  --set hamiCore.enabled=true \
  --set hamiCore.image.repository=howdi2000 \
  --set hamiCore.image.tag=v1.0.0-memlimit4
```

Relevant `values.yaml` knobs:

| key | default | meaning |
|-----|---------|---------|
| `global.gpuSharing` | `true` | enable fractional GPU scheduling (required) |
| `hamiCore.enabled` | `true` | deploy libsync + turn on the binder `hamicore` plugin |
| `hamiCore.image.{repository,name,tag}` | `howdi2000/kai-resource-isolator:v1.0.0-memlimit4` | image carrying `/artifacts/libvgpu.so` |
| `hamiCore.hostLibPath` | `/usr/local/vgpu` | node path for libvgpu.so + ld.so.preload |
| `hamiCore.nodeSelector` | `nvidia.com/gpu.present: "true"` | which nodes to sync onto |

To disable the HAMi integration (plain KAI): `--set hamiCore.enabled=false`.

## Install (raw manifests, no Helm)

`libsync.yaml` in this directory is a standalone version of the DaemonSet + ConfigMap for
`kubectl apply` against a KAI install that already runs the hamicore-enabled binder:

```bash
kubectl apply -f deployments/hami-core/libsync.yaml
```

## Building the images

The KAI service images (admission, binder, …) carry the hamicore plugin code. Build and
push with your registry:

```bash
# legacy docker (no buildx): single-stage distroless image per service
DOCKER_BUILDKIT=0 docker build -f Dockerfile.legacy \
  --build-arg SERVICE_NAME=binder --build-arg TARGETARCH=amd64 \
  -t howdi2000/binder:hami-sm .
docker push howdi2000/binder:hami-sm
# repeat for SERVICE_NAME=admission
```

The `hamiCore.image` (kai-resource-isolator) is the HAMi image that ships
`/artifacts/libvgpu.so`; build it from the KAI-resource-isolator repo.

## Using it

Submit a fractional GPU pod with `schedulerName: kai-scheduler`, a queue label, and the
fraction/SM annotations:

```yaml
metadata:
  labels:
    kai.scheduler/queue: proj-team-a
  annotations:
    gpu-fraction: "0.5"          # 50% of GPU memory
    gpu-sm-percentage: "30"      # 30% SM/compute cap  (or gpu-sm-cores: "24")
spec:
  schedulerName: kai-scheduler
```

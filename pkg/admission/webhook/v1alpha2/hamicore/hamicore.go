// Copyright 2026 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package hamicore

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/kai-scheduler/KAI-scheduler/pkg/binder/common"
	"github.com/kai-scheduler/KAI-scheduler/pkg/binder/common/gpusharingconfigmap"
	"github.com/kai-scheduler/KAI-scheduler/pkg/common/resources"
)

const (
	// vgpuVolumeName is the hostPath volume carrying HAMi-core's libvgpu.so and the
	// ld.so.preload file, distributed to GPU nodes by the libsync DaemonSet.
	vgpuVolumeName = "kai-hami-vgpu"
	// vgpuMountPath must match the libsync DaemonSet install path and the path
	// referenced inside ld.so.preload (HAMi hook layout: /usr/local/vgpu).
	vgpuMountPath = "/usr/local/vgpu"
	ldPreloadPath = "/etc/ld.so.preload"
	ldPreloadKey  = "ld.so.preload"
)

type HamiCore struct{}

func New() *HamiCore {
	return &HamiCore{}
}

func (p *HamiCore) Name() string {
	return "hamicore"
}

func (p *HamiCore) Validate(_ *v1.Pod) error {
	return nil
}

// Mutate injects the CUDA_DEVICE_MEMORY_LIMIT env var (and CUDA_DEVICE_SM_LIMIT when
// a compute cap is requested) into the fractional GPU container. The env vars
// reference the capabilities ConfigMap keys that the hamicore binder plugin
// populates at bind time. The ConfigMap itself and its annotation are set up by the
// gpusharing admission plugin, so this plugin must run after gpusharing.
func (p *HamiCore) Mutate(pod *v1.Pod) error {
	if len(pod.Spec.Containers) == 0 {
		return nil
	}

	if !resources.RequestsGPUFraction(pod) {
		return nil
	}

	containerRef, err := common.GetFractionContainerRef(pod)
	if err != nil {
		return err
	}

	capabilitiesConfigMapName, err := gpusharingconfigmap.ExtractCapabilitiesConfigMapName(pod, containerRef)
	if err != nil {
		return err
	}

	common.AddEnvVarToContainer(containerRef.Container,
		capabilitiesConfigMapEnvVar(common.CudaDeviceMemoryLimit, capabilitiesConfigMapName))

	if resources.RequestsGPUComputeLimit(pod) {
		common.AddEnvVarToContainer(containerRef.Container,
			capabilitiesConfigMapEnvVar(common.CudaDeviceSmLimit, capabilitiesConfigMapName))
	}

	// Inject HAMi-core (libvgpu) so the CUDA_DEVICE_* caps the binder writes are
	// actually enforced: the hostPath volume carries libvgpu.so + ld.so.preload
	// (placed on the node by the libsync DaemonSet), and ld.so.preload LD_PRELOADs
	// libvgpu into the fractional container. Volumes/mounts are immutable after pod
	// creation, so this must happen at admission, not at bind time.
	injectVGPU(pod, containerRef.Container)

	return nil
}

// injectVGPU adds the libvgpu hostPath volume and mounts it (plus ld.so.preload)
// into the fractional container. Idempotent — skips entries already present.
func injectVGPU(pod *v1.Pod, container *v1.Container) {
	hasVolume := false
	for _, v := range pod.Spec.Volumes {
		if v.Name == vgpuVolumeName {
			hasVolume = true
			break
		}
	}
	if !hasVolume {
		pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
			Name: vgpuVolumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: vgpuMountPath,
					Type: ptr.To(v1.HostPathDirectoryOrCreate),
				},
			},
		})
	}

	addMount(container, v1.VolumeMount{Name: vgpuVolumeName, MountPath: vgpuMountPath})
	addMount(container, v1.VolumeMount{Name: vgpuVolumeName, MountPath: ldPreloadPath, SubPath: ldPreloadKey, ReadOnly: true})
}

func addMount(container *v1.Container, mount v1.VolumeMount) {
	for _, m := range container.VolumeMounts {
		if m.Name == mount.Name && m.MountPath == mount.MountPath && m.SubPath == mount.SubPath {
			return
		}
	}
	container.VolumeMounts = append(container.VolumeMounts, mount)
}

// capabilitiesConfigMapEnvVar builds an env var whose value is read from the named
// key of the capabilities ConfigMap, marked optional so the container still starts
// if the binder has not populated the key (e.g. cap not resolvable at bind time).
func capabilitiesConfigMapEnvVar(key, configMapName string) v1.EnvVar {
	return v1.EnvVar{
		Name: key,
		ValueFrom: &v1.EnvVarSource{
			ConfigMapKeyRef: &v1.ConfigMapKeySelector{
				Key: key,
				LocalObjectReference: v1.LocalObjectReference{
					Name: configMapName,
				},
				Optional: ptr.To(true),
			},
		},
	}
}

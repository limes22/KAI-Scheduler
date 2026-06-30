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

	return nil
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

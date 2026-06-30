// Copyright 2026 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package hamicore

import (
	"context"
	"fmt"
	"math"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kai-scheduler/KAI-scheduler/pkg/apis/scheduling/v1alpha2"
	"github.com/kai-scheduler/KAI-scheduler/pkg/binder/common"
	"github.com/kai-scheduler/KAI-scheduler/pkg/binder/plugins/state"
	"github.com/kai-scheduler/KAI-scheduler/pkg/common/constants"
)

type Plugin struct {
	kubeClient client.Client
}

func New(kubeClient client.Client) *Plugin {
	return &Plugin{kubeClient: kubeClient}
}

func (p *Plugin) Name() string {
	return "hamicore"
}

func (p *Plugin) PreBind(
	ctx context.Context, pod *v1.Pod, node *v1.Node, bindRequest *v1alpha2.BindRequest, _ *state.BindingState,
) error {
	if !common.IsSharedGPUAllocation(bindRequest) {
		return nil
	}

	containerRef, err := common.GetFractionContainerRef(pod)
	if err != nil {
		return fmt.Errorf("failed to get fraction container ref: %w", err)
	}

	if cudaDeviceMemoryLimit, err := calculateCudaDeviceMemoryLimit(node, bindRequest); err == nil {
		if err := common.SetCudaDeviceMemoryLimit(ctx, p.kubeClient, pod, containerRef, cudaDeviceMemoryLimit); err != nil {
			return err
		}
	}

	cudaDeviceSmLimit, requested, err := calculateCudaDeviceSmLimit(node, pod)
	if err != nil {
		// A compute cap was requested but cannot be resolved (e.g. core-count mode
		// without the node basis label). Skip enforcement rather than block binding,
		// matching the memory-limit path; the pod runs with shared (uncapped) compute.
		log.FromContext(ctx).Error(err, "skipping CUDA_DEVICE_SM_LIMIT injection",
			"namespace", pod.Namespace, "name", pod.Name)
	} else if requested {
		if err := common.SetCudaDeviceSmLimit(ctx, p.kubeClient, pod, containerRef, cudaDeviceSmLimit); err != nil {
			return err
		}
	}

	return nil
}

func calculateCudaDeviceMemoryLimit(node *v1.Node, bindRequest *v1alpha2.BindRequest) (string, error) {
	if node == nil || bindRequest == nil || bindRequest.Spec.ReceivedGPU == nil {
		return "", fmt.Errorf("missing data for CUDA_DEVICE_MEMORY_LIMIT calculation")
	}

	memoryLabel, found := node.Labels[constants.NvidiaGpuMemory]
	if !found {
		return "", fmt.Errorf("node does not include %s label", constants.NvidiaGpuMemory)
	}

	totalGPUMemoryMib, err := strconv.ParseInt(memoryLabel, 10, 64)
	if err != nil || totalGPUMemoryMib <= 0 {
		return "", fmt.Errorf("invalid %s label value %q", constants.NvidiaGpuMemory, memoryLabel)
	}

	gpuPortion, err := strconv.ParseFloat(bindRequest.Spec.ReceivedGPU.Portion, 64)
	if err != nil || gpuPortion <= 0 {
		return "", fmt.Errorf("invalid received gpu portion %q", bindRequest.Spec.ReceivedGPU.Portion)
	}

	allocatedMemoryMib := int64(float64(totalGPUMemoryMib) * gpuPortion)
	if allocatedMemoryMib <= 0 {
		return "", fmt.Errorf("calculated allocated gpu memory is zero")
	}

	return fmt.Sprintf("%dm", allocatedMemoryMib), nil
}

// calculateCudaDeviceSmLimit resolves the HAMi-core compute cap (CUDA_DEVICE_SM_LIMIT,
// an SM-utilization percentage 1-100) from the pod's compute-request annotations.
// Two request modes are supported, gpu-sm-percentage taking precedence over gpu-sm-cores:
//   - gpu-sm-percentage: used directly as the SM-limit percentage.
//   - gpu-sm-cores: an absolute SM/core count, converted to a percentage using the
//     nvidia.com/gpu.cores node label (the per-GPU SM count on the bound node).
//
// The bool return reports whether a compute cap was requested at all; when false the
// value is empty and no env should be set.
func calculateCudaDeviceSmLimit(node *v1.Node, pod *v1.Pod) (string, bool, error) {
	if pod == nil || pod.Annotations == nil {
		return "", false, nil
	}

	if pctStr, found := pod.Annotations[constants.GpuSmPercentage]; found && pctStr != "" {
		pct, err := strconv.Atoi(pctStr)
		if err != nil || pct <= 0 || pct > 100 {
			return "", true, fmt.Errorf("invalid %s annotation value %q (expected 1-100)",
				constants.GpuSmPercentage, pctStr)
		}
		return strconv.Itoa(pct), true, nil
	}

	if coresStr, found := pod.Annotations[constants.GpuSmCores]; found && coresStr != "" {
		cores, err := strconv.ParseInt(coresStr, 10, 64)
		if err != nil || cores <= 0 {
			return "", true, fmt.Errorf("invalid %s annotation value %q", constants.GpuSmCores, coresStr)
		}
		if node == nil {
			return "", true, fmt.Errorf("missing node for core-based SM limit calculation")
		}
		coresLabel, found := node.Labels[constants.NvidiaGpuCores]
		if !found {
			return "", true, fmt.Errorf("node does not include %s label; cannot convert core count to SM percentage",
				constants.NvidiaGpuCores)
		}
		totalCores, err := strconv.ParseInt(coresLabel, 10, 64)
		if err != nil || totalCores <= 0 {
			return "", true, fmt.Errorf("invalid %s label value %q", constants.NvidiaGpuCores, coresLabel)
		}
		// Round up so a sub-core request still yields a usable (>=1%) cap, and clamp to 100.
		pct := int64(math.Ceil(float64(cores) / float64(totalCores) * 100))
		if pct < 1 {
			pct = 1
		}
		if pct > 100 {
			pct = 100
		}
		return strconv.FormatInt(pct, 10), true, nil
	}

	return "", false, nil
}

func (p *Plugin) PostBind(
	context.Context, *v1.Pod, *v1.Node, *v1alpha2.BindRequest, *state.BindingState,
) {
}

func (p *Plugin) Rollback(
	context.Context, *v1.Pod, *v1.Node, *v1alpha2.BindRequest, *state.BindingState,
) error {
	return nil
}

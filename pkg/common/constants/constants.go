// Copyright 2025 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package constants

import "time"

const (
	AppLabelName      = "app"
	NvidiaGpuResource = "nvidia.com/gpu"
	NvidiaGpuMemory   = "nvidia.com/gpu.memory"
	// NvidiaGpuCores is the per-GPU SM/core count advertised on GPU nodes. Used as
	// the basis to convert a core-count compute request into an SM-limit percentage.
	NvidiaGpuCores            = "nvidia.com/gpu.cores"
	GpuResource               = "gpu"
	UnlimitedResourceQuantity = float64(-1)

	DefaultQueuePriority                  = 100
	DefaultPodGroupPriority               = 50 // Default when no global default priority exists
	DefaultNodePoolName                   = "default"
	DefaultMetricsNamespace               = "kai"
	DefaultQueueLabel                     = "kai.scheduler/queue"
	DefaultSchedulerName                  = "kai-scheduler"
	DefaultKAINamespace                   = "kai-scheduler"
	DefaultResourceReservationName        = "kai-resource-reservation"
	DefaultScaleAdjustName                = "kai-scale-adjust"
	DefaultKAIConfigSingeltonInstanceName = "kai-config"
	DefaultNodePoolLabelKey               = "kai.scheduler/node-pool"
	DefaultRuntimeClassName               = "nvidia"
	DefaultStuckInReleasingThreshold      = 2 * time.Minute

	DefaultCPUWorkerNodeLabelKey = "node-role.kubernetes.io/cpu-worker"
	DefaultGPUWorkerNodeLabelKey = "node-role.kubernetes.io/gpu-worker"
	DefaultMIGWorkerNodeLabelKey = "node-role.kubernetes.io/mig-enabled"

	// Pod Groups
	PodGrouperWarning   = "PodGrouperWarning"
	TopOwnerMetadataKey = "kai.scheduler/top-owner-metadata"

	// Annotations
	PodGroupAnnotationForPod = "pod-group-name"
	SkipPodGrouperAnnotation = "kai.scheduler/skip-podgrouper"
	GpuFraction              = "gpu-fraction"
	GpuFractionContainerName = "gpu-fraction-container-name"
	GpuMemory                = "gpu-memory"
	// GpuSmPercentage caps GPU compute as a percentage (1-100) of a single GPU's SMs.
	GpuSmPercentage = "gpu-sm-percentage"
	// GpuSmCores caps GPU compute as an absolute SM/core count, converted to a
	// percentage at bind time using the nvidia.com/gpu.cores node label.
	GpuSmCores                    = "gpu-sm-cores"
	ReceivedResourceType          = "received-resource-type"
	GpuFractionsNumDevices        = "gpu-fraction-num-devices"
	MpsAnnotation                 = "mps"
	StalePodgroupTimeStamp        = "kai.scheduler/stale-podgroup-timestamp"
	LastStartTimeStamp            = "kai.scheduler/last-start-timestamp"
	GpuSharingConfigMapAnnotation = "runai/shared-gpu-configmap"
	NvidiaVisibleDevices          = "NVIDIA_VISIBLE_DEVICES"

	// UsageDB Prometheus Selector
	DefaultAccountingLabelKey   = "kai.scheduler/accounting"
	DefaultAccountingLabelValue = "true"

	// Labels
	GPUGroup                 = "runai-gpu-group"
	MultiGpuGroupLabelPrefix = GPUGroup + "/"
	MigStrategyLabel         = "nvidia.com/mig.strategy"
	GpuCountLabel            = "nvidia.com/gpu.count"
	SubGroupLabelKey         = "kai.scheduler/subgroup-name"
)

// QueueValidatedVersions returns the list of queue versions that we validate with a webhook. This will be used by the
// kai operator when installing webhooks. When changing this, test for backwards compatibility.
func QueueValidatedVersions() []string {
	return []string{"v2"}
}

// PodGroupValidatedVersions returns the list of podgroup versions that we validate with a webhook.
// This will be used by the kai operator when installing webhooks. When changing this, test for backwards compatibility.
func PodGroupValidatedVersions() []string {
	return []string{"v2alpha2"}
}

// Copyright 2025 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package common

const (
	GPUPortion            = "GPU_PORTION"
	CudaDeviceMemoryLimit = "CUDA_DEVICE_MEMORY_LIMIT"
	// CudaDeviceSmLimit is the HAMi-core (libvgpu) compute-limit env var: the
	// percentage (1-100) of streaming-multiprocessor utilization the pod may use.
	CudaDeviceSmLimit    = "CUDA_DEVICE_SM_LIMIT"
	ReceivedTypeFraction = "Fraction"
	ReceivedTypeRegular  = "Regular"
)

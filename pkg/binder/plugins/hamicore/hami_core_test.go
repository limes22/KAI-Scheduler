// Copyright 2026 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package hamicore

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kai-scheduler/KAI-scheduler/pkg/common/constants"
)

func nodeWithCores(cores string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{constants.NvidiaGpuCores: cores},
		},
	}
}

func podWithAnnotations(annotations map[string]string) *v1.Pod {
	return &v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}}
}

func TestCalculateCudaDeviceSmLimit(t *testing.T) {
	tests := []struct {
		name          string
		node          *v1.Node
		pod           *v1.Pod
		wantValue     string
		wantRequested bool
		wantErr       bool
	}{
		{
			name:          "no compute annotation",
			node:          nodeWithCores("100"),
			pod:           podWithAnnotations(map[string]string{constants.GpuFraction: "0.5"}),
			wantRequested: false,
		},
		{
			name:          "percentage used directly",
			node:          nodeWithCores("100"),
			pod:           podWithAnnotations(map[string]string{constants.GpuSmPercentage: "30"}),
			wantValue:     "30",
			wantRequested: true,
		},
		{
			name:          "percentage takes precedence over cores",
			node:          nodeWithCores("100"),
			pod:           podWithAnnotations(map[string]string{constants.GpuSmPercentage: "25", constants.GpuSmCores: "80"}),
			wantValue:     "25",
			wantRequested: true,
		},
		{
			name:          "percentage out of range",
			node:          nodeWithCores("100"),
			pod:           podWithAnnotations(map[string]string{constants.GpuSmPercentage: "150"}),
			wantRequested: true,
			wantErr:       true,
		},
		{
			name:          "percentage non-numeric",
			node:          nodeWithCores("100"),
			pod:           podWithAnnotations(map[string]string{constants.GpuSmPercentage: "abc"}),
			wantRequested: true,
			wantErr:       true,
		},
		{
			name:          "cores converted to percentage",
			node:          nodeWithCores("128"),
			pod:           podWithAnnotations(map[string]string{constants.GpuSmCores: "32"}),
			wantValue:     "25",
			wantRequested: true,
		},
		{
			name:          "cores rounded up to at least one percent",
			node:          nodeWithCores("1000"),
			pod:           podWithAnnotations(map[string]string{constants.GpuSmCores: "1"}),
			wantValue:     "1",
			wantRequested: true,
		},
		{
			name:          "cores clamped to one hundred percent",
			node:          nodeWithCores("128"),
			pod:           podWithAnnotations(map[string]string{constants.GpuSmCores: "256"}),
			wantValue:     "100",
			wantRequested: true,
		},
		{
			name:          "cores without node label errors",
			node:          &v1.Node{},
			pod:           podWithAnnotations(map[string]string{constants.GpuSmCores: "32"}),
			wantRequested: true,
			wantErr:       true,
		},
		{
			name:          "cores invalid value errors",
			node:          nodeWithCores("128"),
			pod:           podWithAnnotations(map[string]string{constants.GpuSmCores: "-5"}),
			wantRequested: true,
			wantErr:       true,
		},
		{
			name:          "cores with invalid node label errors",
			node:          nodeWithCores("not-a-number"),
			pod:           podWithAnnotations(map[string]string{constants.GpuSmCores: "32"}),
			wantRequested: true,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, requested, err := calculateCudaDeviceSmLimit(tt.node, tt.pod)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if requested != tt.wantRequested {
				t.Errorf("requested = %v, want %v", requested, tt.wantRequested)
			}
			if !tt.wantErr && value != tt.wantValue {
				t.Errorf("value = %q, want %q", value, tt.wantValue)
			}
		})
	}
}

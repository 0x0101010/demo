package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrintReason(t *testing.T) {
	tests := []struct {
		pod    apiv1.Pod
		expect string
	}{
		{
			// Test name, num of containers, restarts, container ready status
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test1"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Phase: "podPhase",
					ContainerStatuses: []apiv1.ContainerStatus{
						{Ready: true, RestartCount: 3, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
						{RestartCount: 3},
					},
				},
			},
			"podPhase",
		},
		{
			// Test container error overwrites pod phase
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test2"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Phase: "podPhase",
					ContainerStatuses: []apiv1.ContainerStatus{
						{Ready: true, RestartCount: 3, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
						{State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{Reason: "ContainerWaitingReason"}}, RestartCount: 3},
					},
				},
			},
			"ContainerWaitingReason",
		},
		{
			// Test the same as the above but with Terminated state and the first container overwrites the rest
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test3"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Phase: "podPhase",
					ContainerStatuses: []apiv1.ContainerStatus{
						{State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{Reason: "ContainerWaitingReason"}}, RestartCount: 3},
						{State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{Reason: "ContainerTerminatedReason"}}, RestartCount: 3},
					},
				},
			},
			"ContainerWaitingReason",
		},
		{
			// Test ready is not enough for reporting running
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test4"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Phase: "podPhase",
					ContainerStatuses: []apiv1.ContainerStatus{
						{Ready: true, RestartCount: 3, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
						{Ready: true, RestartCount: 3},
					},
				},
			},
			"podPhase",
		},
		{
			// Test ready is not enough for reporting running
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test5"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Reason: "podReason",
					Phase:  "podPhase",
					ContainerStatuses: []apiv1.ContainerStatus{
						{Ready: true, RestartCount: 3, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
						{Ready: true, RestartCount: 3},
					},
				},
			},
			"podReason",
		},
		{
			// Test pod has 2 containers, one is running and the other is completed, w/o ready condition
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test6"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Phase:  "Running",
					Reason: "",
					ContainerStatuses: []apiv1.ContainerStatus{
						{Ready: true, RestartCount: 3, State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{Reason: "Completed", ExitCode: 0}}},
						{Ready: true, RestartCount: 3, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
					},
				},
			},
			"NotReady",
		},
		{
			// Test pod has 2 containers, one is running and the other is completed, with ready condition
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test6"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Phase:  "Running",
					Reason: "",
					ContainerStatuses: []apiv1.ContainerStatus{
						{Ready: true, RestartCount: 3, State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{Reason: "Completed", ExitCode: 0}}},
						{Ready: true, RestartCount: 3, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
					},
					Conditions: []apiv1.PodCondition{
						{Type: apiv1.PodReady, Status: apiv1.ConditionTrue, LastProbeTime: metav1.Time{Time: time.Now()}},
					},
				},
			},
			"Running",
		},
		{
			// Test pod has 1 init container restarting and 1 container not running
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test7"},
				Spec:       apiv1.PodSpec{InitContainers: make([]apiv1.Container, 1), Containers: make([]apiv1.Container, 1)},
				Status: apiv1.PodStatus{
					Phase: "podPhase",
					InitContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready:                false,
							RestartCount:         3,
							State:                apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-10 * time.Second))}},
						},
					},
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready:        false,
							RestartCount: 0,
							State:        apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{}},
						},
					},
				},
			},
			"Init:0/1",
		},
		{
			// Test pod has 2 init containers, one restarting and the other not running, and 1 container not running
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test8"},
				Spec:       apiv1.PodSpec{InitContainers: make([]apiv1.Container, 2), Containers: make([]apiv1.Container, 1)},
				Status: apiv1.PodStatus{
					Phase: "podPhase",
					InitContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready:                false,
							RestartCount:         3,
							State:                apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-10 * time.Second))}},
						},
						{
							Ready: false,
							State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{}},
						},
					},
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready: false,
							State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{}},
						},
					},
				},
			},
			"Init:0/2",
		},
		{
			// Test pod has 2 init containers, one completed without restarts and the other restarting, and 1 container not running
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test9"},
				Spec:       apiv1.PodSpec{InitContainers: make([]apiv1.Container, 2), Containers: make([]apiv1.Container, 1)},
				Status: apiv1.PodStatus{
					Phase: "podPhase",
					InitContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready: false,
							State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{}},
						},
						{
							Ready:                false,
							RestartCount:         3,
							State:                apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-10 * time.Second))}},
						},
					},
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready: false,
							State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{}},
						},
					},
				},
			},
			"Init:1/2",
		},
		{
			// Test pod has 2 init containers, one completed with restarts and the other restarting, and 1 container not running
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test10"},
				Spec:       apiv1.PodSpec{InitContainers: make([]apiv1.Container, 2), Containers: make([]apiv1.Container, 1)},
				Status: apiv1.PodStatus{
					Phase: "podPhase",
					InitContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready:                false,
							RestartCount:         2,
							State:                apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-2 * time.Minute))}},
						},
						{
							Ready:                false,
							RestartCount:         3,
							State:                apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-10 * time.Second))}},
						},
					},
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready: false,
							State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{}},
						},
					},
				},
			},
			"Init:1/2",
		},
		{
			// Test pod has 1 init container completed with restarts and one container restarting
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test11"},
				Spec:       apiv1.PodSpec{InitContainers: make([]apiv1.Container, 1), Containers: make([]apiv1.Container, 1)},
				Status: apiv1.PodStatus{
					Phase: "Running",
					InitContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready:                false,
							RestartCount:         2,
							State:                apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-2 * time.Minute))}},
						},
					},
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready:                false,
							RestartCount:         4,
							State:                apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-20 * time.Second))}},
						},
					},
				},
			},
			"Running",
		},
		{
			// Test pod has 1 container that restarted 5d ago
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test12"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 1)},
				Status: apiv1.PodStatus{
					Phase: "Running",
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready:                true,
							RestartCount:         3,
							State:                apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-5 * 24 * time.Hour))}},
						},
					},
				},
			},
			"Running",
		},
		{
			// Test pod has 2 containers, one has never restarted and the other has restarted 10d ago
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test13"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Phase: "Running",
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready:        true,
							RestartCount: 0,
							State:        apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
						},
						{
							Ready:                true,
							RestartCount:         3,
							State:                apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-10 * 24 * time.Hour))}},
						},
					},
				},
			},
			"Running",
		},
		{
			// Test pod has 2 containers, one restarted 5d ago and the other restarted 20d ago
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test14"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Phase: "Running",
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Ready:                true,
							RestartCount:         6,
							State:                apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-5 * 24 * time.Hour))}},
						},
						{
							Ready:                true,
							RestartCount:         3,
							State:                apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
							LastTerminationState: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(-20 * 24 * time.Hour))}},
						},
					},
				},
			},
			"Running",
		},
		{
			// Test PodScheduled condition with reason WaitingForGates
			apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test15"},
				Spec:       apiv1.PodSpec{Containers: make([]apiv1.Container, 2)},
				Status: apiv1.PodStatus{
					Phase: "podPhase",
					Conditions: []apiv1.PodCondition{
						{
							Type:   apiv1.PodScheduled,
							Status: apiv1.ConditionFalse,
							Reason: apiv1.PodReasonSchedulingGated,
						},
					},
				},
			},
			apiv1.PodReasonSchedulingGated,
		},
	}

	for i, test := range tests {
		reason := printReason(&test.pod)
		if !reflect.DeepEqual(test.expect, reason) {
			t.Errorf("%d mismatch: %s", i, cmp.Diff(test.expect, reason))
		}
	}
}

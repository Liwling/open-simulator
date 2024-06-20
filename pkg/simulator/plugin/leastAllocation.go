package plugin

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	resourcehelper "k8s.io/kubectl/pkg/util/resource"
	framework "k8s.io/kubernetes/pkg/scheduler/framework"

	simontype "github.com/alibaba/open-simulator/pkg/type"
)

// LeastAllocationPlugin is a score plugin for scheduling framework
type LeastAllocationPlugin struct {
	nodesInfo map[string]*framework.NodeInfo
}

var _ = framework.ScorePlugin(&LeastAllocationPlugin{})

func NewLeastAllocationPlugin(nodesInfo map[string]*framework.NodeInfo, configuration runtime.Object, f framework.Handle) (framework.Plugin, error) {
	return &LeastAllocationPlugin{
		nodesInfo: nodesInfo,
	}, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (plugin *LeastAllocationPlugin) Name() string {
	return simontype.LeastAllocationPluginName
}

// Score invoked at the score extension point.
func (plugin *LeastAllocationPlugin) Score(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) (int64, *framework.Status) {
	podReq, _ := resourcehelper.PodRequestsAndLimits(pod)
	currentNodeInfo := plugin.nodesInfo[nodeName]

	requestedResourceCopy := currentNodeInfo.Requested.Clone()
	requestedResourceCopy.Add(podReq)

	cpuCapacity, cpuRequested := currentNodeInfo.Allocatable.MilliCPU, requestedResourceCopy.MilliCPU
	memoryCapacity, memoryRequested := currentNodeInfo.Allocatable.Memory, requestedResourceCopy.Memory
	cpuScore := ((cpuCapacity - cpuRequested) * framework.MaxNodeScore) / cpuCapacity
	memoryScore := ((memoryCapacity - memoryRequested) * framework.MaxNodeScore) / memoryCapacity
	score := (cpuScore + memoryScore) / 2

	fmt.Printf("\ncpuScore: %d memScore: %d nodename: %s score: %d\n", cpuScore, memoryScore, nodeName, score)

	return score, framework.NewStatus(framework.Success)
}

// ScoreExtensions of the Score plugin.
func (plugin *LeastAllocationPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

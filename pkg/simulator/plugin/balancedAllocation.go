package plugin

import (
	"context"
	"math"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	resourcehelper "k8s.io/kubectl/pkg/util/resource"
	framework "k8s.io/kubernetes/pkg/scheduler/framework"

	simontype "github.com/alibaba/open-simulator/pkg/type"
)

// BalancedAllocationPlugin is a score plugin for scheduling framework
type BalancedAllocationPlugin struct {
	nodesInfo map[string]*framework.NodeInfo
}

var _ = framework.ScorePlugin(&BalancedAllocationPlugin{})

func NewBalancedAllocationPlugin(nodesInfo map[string]*framework.NodeInfo, configuration runtime.Object, f framework.Handle) (framework.Plugin, error) {
	return &BalancedAllocationPlugin{
		nodesInfo: nodesInfo,
	}, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (plugin *BalancedAllocationPlugin) Name() string {
	return simontype.BalancedAllocationPluginName
}

// Score invoked at the score extension point.
func (plugin *BalancedAllocationPlugin) Score(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) (int64, *framework.Status) {
	podReq, _ := resourcehelper.PodRequestsAndLimits(pod)
	currentNodeInfo := plugin.nodesInfo[nodeName]

	requestedResourceCopy := currentNodeInfo.Requested.Clone()
	requestedResourceCopy.Add(podReq)
	postCpuPercentage := float64(requestedResourceCopy.MilliCPU) / float64(currentNodeInfo.Allocatable.MilliCPU)
	postMemoryPercentage := float64(requestedResourceCopy.Memory) / float64(currentNodeInfo.Allocatable.Memory)
	postDiff := math.Abs(postCpuPercentage - postMemoryPercentage)

	return int64((1 - postDiff) * float64(framework.MaxNodeScore)), framework.NewStatus(framework.Success)
}

// ScoreExtensions of the Score plugin.
func (plugin *BalancedAllocationPlugin) ScoreExtensions() framework.ScoreExtensions {
	return plugin
}

// NormalizeScore invoked after scoring all nodes.
func (plugin *BalancedAllocationPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, scores framework.NodeScoreList) *framework.Status {
	// Find highest and lowest scores.
	var highest int64 = -math.MaxInt64
	var lowest int64 = math.MaxInt64
	for _, nodeScore := range scores {
		if nodeScore.Score > highest {
			highest = nodeScore.Score
		}
		if nodeScore.Score < lowest {
			lowest = nodeScore.Score
		}
	}

	// Transform the highest to lowest score range to fit the framework's min to max node score range.
	oldRange := highest - lowest
	newRange := framework.MaxNodeScore - framework.MinNodeScore
	for i, nodeScore := range scores {
		if oldRange == 0 {
			scores[i].Score = framework.MinNodeScore
		} else {
			scores[i].Score = ((nodeScore.Score - lowest) * newRange / oldRange) + framework.MinNodeScore
		}
	}

	return framework.NewStatus(framework.Success)
}

package algo

import (
	"fmt"
	"github.com/alibaba/open-simulator/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sort"
)

var nodeResource v1.ResourceList

type balanceQueue struct {
	pods         []*v1.Pod
	balanceValue []float64
}

func SortPodsBasedOnDiff(node *v1.Node, pods []*v1.Pod) {
	queue := balanceQueue{pods: pods, balanceValue: make([]float64, len(pods))}

	for i := range pods {
		newNodeInfo := framework.NewNodeInfo(pods[i])
		_ = newNodeInfo.SetNode(node)
		queue.balanceValue[i], _ = utils.CalculateBalanceValue(newNodeInfo.Requested, newNodeInfo.Allocatable)
	}
	sort.Sort(queue)
	fmt.Println(queue.balanceValue)
}

func (b balanceQueue) Len() int {
	return len(b.pods)
}

func (b balanceQueue) Swap(i int, j int) {
	b.pods[i], b.pods[j] = b.pods[j], b.pods[i]
	b.balanceValue[i], b.balanceValue[j] = b.balanceValue[j], b.balanceValue[i]
}

func (b balanceQueue) Less(i int, j int) bool {
	if b.balanceValue[i] < b.balanceValue[j] {
		return true
	}
	return false
}

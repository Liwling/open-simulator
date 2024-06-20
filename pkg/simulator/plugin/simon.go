package plugin

import (
	"context"
	"fmt"
	simontype "github.com/alibaba/open-simulator/pkg/type"
	"github.com/alibaba/open-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	externalclientset "k8s.io/client-go/kubernetes"
	resourcehelper "k8s.io/kubectl/pkg/util/resource"
	framework "k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/helper"
)

// SimonPlugin is a plugin for scheduling framework
type SimonPlugin struct {
	fakeclient externalclientset.Interface
	nodesInfo  map[string]*framework.NodeInfo
}

var _ = framework.ScorePlugin(&SimonPlugin{})
var _ = framework.BindPlugin(&SimonPlugin{})

func NewSimonPlugin(fakeclient externalclientset.Interface, nodesInfo map[string]*framework.NodeInfo, configuration runtime.Object, f framework.Handle) (framework.Plugin, error) {
	return &SimonPlugin{
		fakeclient: fakeclient,
		nodesInfo:  nodesInfo,
	}, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (plugin *SimonPlugin) Name() string {
	return simontype.SimonPluginName
}

// Bind invoked at the bind extension point.
func (plugin *SimonPlugin) Bind(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) *framework.Status {
	return plugin.BindPodToNode(ctx, state, pod, nodeName)
}

// Score invoked at the score extension point.
func (plugin *SimonPlugin) Score(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) (int64, *framework.Status) {
	var score int64
	offset := 0.5
	podReq, _ := resourcehelper.PodRequestsAndLimits(pod)
	if len(podReq) == 0 {
		score = int64((0 + offset) * float64(framework.MaxNodeScore))
		return score, framework.NewStatus(framework.Success)
	}

	//currentBalanceValue := utils.CalculateBalanceValue(plugin.nodesInfo[nodeName].Requested, plugin.nodesInfo[nodeName].Allocatable)
	//requestedResourceCopy := plugin.nodesInfo[nodeName].Requested.Clone()
	//requestedResourceCopy.Add(podReq)
	//postBalanceValue := utils.CalculateBalanceValue(requestedResourceCopy, plugin.nodesInfo[nodeName].Allocatable)
	//diff := postBalanceValue - currentBalanceValue

	currentPercentage := utils.CalculateResourcePercentage(plugin.nodesInfo[nodeName].Requested, plugin.nodesInfo[nodeName].Allocatable)
	requestedResourceCopy := plugin.nodesInfo[nodeName].Requested.Clone()
	requestedResourceCopy.Add(podReq)
	postPercentage := utils.CalculateResourcePercentage(requestedResourceCopy, plugin.nodesInfo[nodeName].Allocatable)
	postPercentageMaxDifference := utils.CalculateResourceMaxDifference(postPercentage)
	cur, post := utils.CalculateResourceValue(currentPercentage), utils.CalculateResourceValue(postPercentage)

	//w, score1, score2 := 0.2, int64(0), int64(0)
	//if diff > 0 {
	//	p := math.Abs(diff) / (1 - currentBalanceValue)
	//	score1, score2 = int64(math.Round(1/w*p*math.Abs(diff)*postBalanceValue*100)), int64(math.Round((1-math.Abs(diff))*postBalanceValue*100))
	//} else {
	//	score1, score2 = 0, int64(math.Round(postBalanceValue*100))
	//}
	//

	flag := float64(0)
	threshold := 0.2
	if postPercentageMaxDifference < threshold {
		flag = 1
	}

	diff := post - cur
	if diff <= 0 {
		score = int64((0.5+diff/2)*100 - (1-cur)*10 + flag*100)
	} else {
		score = int64((0.5+diff/2)*100 + (1-cur)*10 + flag*100)
	}

	//w, score1, score2 := float64(0), int64(0), int64(0)
	//if diff > 0 {
	//	w = math.Abs(diff) / (1 - currentBalanceValue) * 10
	//	score1, score2 = int64((1+diff)*100), int64((1-currentBalanceValue)*10*w)
	//	score = int64((1+diff)*100) + int64((1-currentBalanceValue)*10*w)
	//} else {
	//	w = math.Abs(diff) / currentBalanceValue * 10
	//	score1, score2 = int64((1+diff)*100), int64((1-currentBalanceValue)*10*w)
	//	score = int64((1+diff)*100) - int64((1-currentBalanceValue)*10*w)
	//}

	//w := 0.2
	//if diff > 0 {
	//	if diff >= w {
	//		score = int64((1-currentBalanceValue)*(1+diff)*100) * 100
	//	} else if diff < 0.1 {
	//		score = int64(postBalanceValue * (1 + diff) * 100)
	//	} else {
	//		score = int64(postBalanceValue*(1+diff)*100) * 10
	//	}
	//} else {
	//	if math.Abs(diff) >= w {
	//		score = int64(currentBalanceValue * (1 + diff) * 10)
	//	} else {
	//		score = int64(postBalanceValue * (1 + diff) * 100)
	//	}
	//}

	//fmt.Printf("\ncurrent: %.3f post: %.3f diff: %.3f nodename: %s score: %d\n", currentPercentage, postPercentage, diff, nodeName, score)
	fmt.Printf("\ncurrent: %.3f post: %.3f postPercentageMaxDiff: %.3f nodename: %s score: %d\n", cur, post, postPercentageMaxDifference, nodeName, score)
	return score, framework.NewStatus(framework.Success)
}

// ScoreExtensions of the Score plugin.
func (plugin *SimonPlugin) ScoreExtensions() framework.ScoreExtensions {
	return plugin
}

//NormalizeScore invoked after scoring all nodes.
//func (plugin *SimonPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, scores framework.NodeScoreList) *framework.Status {
//	return NormalizeScore(scores, plugin.nodesInfo)
//}

func (plugin *SimonPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, scores framework.NodeScoreList) *framework.Status {
	return helper.DefaultNormalizeScore(framework.MaxNodeScore, false, scores)
}

//func NormalizeScore(maxPriority int64, reverse bool, scores framework.NodeScoreList) *framework.Status {
//	var maxCount int64
//	for i := range scores {
//		if scores[i].Score > maxCount {
//			maxCount = scores[i].Score
//		}
//	}
//
//	if maxCount == 0 {
//		if reverse {
//			for i := range scores {
//				scores[i].Score = maxPriority
//			}
//		}
//		return nil
//	}
//
//	for i := range scores {
//		score := scores[i].Score
//		fmt.Printf(" |%d ", score)
//
//		score = maxPriority * score / maxCount
//		if reverse {
//			score = maxPriority - score
//		}
//
//		scores[i].Score = score
//		fmt.Printf("%d", score)
//	}
//
//	fmt.Println()
//	return nil
//}
//

//func NormalizeScore(scores framework.NodeScoreList, nodesInfo map[string]*framework.NodeInfo) *framework.Status {
//	maxScoreNode := make([]int, 0)
//	maxScore := int64(0)
//	for number := range scores {
//		if scores[number].Score > maxScore {
//			maxScore = scores[number].Score
//			maxScoreNode = []int{number}
//		} else if scores[number].Score == maxScore {
//			maxScoreNode = append(maxScoreNode, number)
//		}
//	}
//
//	if len(maxScoreNode) > 1 {
//		if maxScore <= int64(50) {
//			maxBalanceValue, number := float64(0), -1
//			for _, nodeNumber := range maxScoreNode {
//				nodeBalanceValue := utils.CalculateBalanceValue(nodesInfo[scores[nodeNumber].Name].Requested, nodesInfo[scores[nodeNumber].Name].Allocatable)
//				if nodeBalanceValue > maxBalanceValue {
//					maxBalanceValue = nodeBalanceValue
//					number = nodeNumber
//				}
//			}
//
//			scores[number].Score++
//		} else {
//			minBalanceValue, number := float64(1), -1
//			for _, nodeNumber := range maxScoreNode {
//				nodeBalanceValue := utils.CalculateBalanceValue(nodesInfo[scores[nodeNumber].Name].Requested, nodesInfo[scores[nodeNumber].Name].Allocatable)
//				if nodeBalanceValue < minBalanceValue {
//					minBalanceValue = nodeBalanceValue
//					number = nodeNumber
//				}
//			}
//
//			scores[number].Score++
//		}
//	}
//
//	fmt.Println(scores)
//	return nil
//}

// BindPodToNode bind pod to a node and trigger pod update event
func (plugin *SimonPlugin) BindPodToNode(ctx context.Context, state *framework.CycleState, p *corev1.Pod, nodeName string) *framework.Status {
	// fmt.Printf("bind pod %s/%s to node %s\n", p.Namespace, p.Name, nodeName)
	// Step 1: update pod info
	pod, err := plugin.fakeclient.CoreV1().Pods(p.Namespace).Get(context.TODO(), p.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("fake get error %v", err)
		return framework.NewStatus(framework.Error, fmt.Sprintf("Unable to bind: %v", err))
	}
	updatedPod := pod.DeepCopy()
	updatedPod.Spec.NodeName = nodeName
	updatedPod.Status.Phase = corev1.PodRunning

	// Step 2: update pod
	// here assuming the pod is already in the resource storage
	// so the update is needed to emit update event in case a handler is registered
	_, err = plugin.fakeclient.CoreV1().Pods(pod.Namespace).Update(context.TODO(), updatedPod, metav1.UpdateOptions{})
	if err != nil {
		log.Errorf("fake update error %v", err)
		return framework.NewStatus(framework.Error, fmt.Sprintf("Unable to add new pod: %v", err))
	}

	return nil
}

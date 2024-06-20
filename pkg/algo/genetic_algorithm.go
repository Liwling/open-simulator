package algo

import (
	"fmt"
	"github.com/alibaba/open-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"math/rand"
)

type individual struct {
	podsCode []int
	fitness  float64
	flag     int
}

func GeneticAlgorithm(popSize int, mutationRate float64, iteration int, pods []*corev1.Pod, nodes []*corev1.Node) int {
	organism := createOrganism(popSize, pods, nodes)
	if len(organism) == 0 {
		return -1
	}

	bestIndividual := getBestIndividual(organism)
	bestIndividual0 := getBestAndFeasibleOrganism(organism)

	fmt.Println("Original organism")
	for i := range organism {
		fmt.Printf("%.3f ", organism[i].fitness)
	}
	fmt.Println()

	for generation := 0; generation < iteration; generation++ {
		pool := createPool(organism, bestIndividual.fitness)
		organism = natureSelect(popSize, mutationRate, pods, nodes, pool)

		bestIndividual = getBestIndividual(organism)
		bestIndividual0 = getBestAndFeasibleOrganism(organism)
		fmt.Printf("%.3f\n", bestIndividual0.fitness)
	}

	fmt.Println("Final organism")
	for i := range organism {
		fmt.Printf("%.3f ", organism[i].fitness)
	}
	fmt.Println()

	if bestIndividual0.fitness == 0 {
		return -1
	}

	for i, podCode := range bestIndividual.podsCode {
		pods[i].Spec.NodeName = nodes[podCode].Name
	}

	return 0
}

func initNodeInfo(nodes []*corev1.Node) []*framework.NodeInfo {
	nodesInfo := make([]*framework.NodeInfo, len(nodes))

	for i := range nodesInfo {
		newNodeInfo := framework.NewNodeInfo()
		err := newNodeInfo.SetNode(nodes[i])
		if err != nil {
			log.Errorf("failed to initial information of node: %v\n", err)
		}

		nodesInfo[i] = newNodeInfo
	}

	return nodesInfo
}

func createIndividual(podNum int, nodeNum int) individual {
	podCode := make([]int, podNum)
	for i := range podCode {
		podCode[i] = rand.Intn(nodeNum)
	}

	idv := individual{
		podsCode: podCode,
	}

	return idv
}

func createOrganism(popSize int, pods []*corev1.Pod, nodes []*corev1.Node) []individual {
	podNum, nodeNum := len(pods), len(nodes)
	organism := make([]individual, popSize)

	for population := 0; population < popSize; {
		newIndividual := createIndividual(podNum, nodeNum)
		fitness, flag := calculateIndividualFitness(newIndividual, pods, nodes)

		if flag == 1 {
			newIndividual.fitness = fitness / 2
		} else {
			newIndividual.fitness = fitness
		}

		newIndividual.flag = flag
		organism[population] = newIndividual
		population++
	}

	return organism
}

func calculateIndividualFitness(idv individual, pods []*corev1.Pod, nodes []*corev1.Node) (float64, int) {
	nodesInfo := initNodeInfo(nodes)

	for i, podCode := range idv.podsCode {
		nodesInfo[podCode].AddPod(pods[i])
	}

	clusterBalanceValue, flag0 := float64(0), 0
	for i := range nodesInfo {
		nodeBalanceValue, flag := utils.CalculateBalanceValue(nodesInfo[i].Requested, nodesInfo[i].Allocatable)
		if flag == 1 {
			flag0 = 1
		}
		clusterBalanceValue += nodeBalanceValue
	}

	return clusterBalanceValue / float64(len(nodes)), flag0
}

func getBestIndividual(organism []individual) individual {
	maxFitness, idvNumber := float64(0), -1
	for i := range organism {
		if organism[i].fitness > maxFitness {
			maxFitness = organism[i].fitness
			idvNumber = i
		}
	}

	return organism[idvNumber]
}

func getBestAndFeasibleOrganism(organism []individual) individual {
	maxFitness, idvNumber := float64(0), -1
	for i := range organism {
		if organism[i].fitness > maxFitness && organism[i].flag == 0 {
			maxFitness = organism[i].fitness
			idvNumber = i
		}
	}

	if idvNumber == -1 {
		return individual{}
	}

	return organism[idvNumber]
}

func createPool(organism []individual, maxFitness float64) []individual {
	pool := make([]individual, 0)

	for i := range organism {
		num := int((organism[i].fitness / maxFitness) * 50)
		for j := 0; j < num; j++ {
			pool = append(pool, organism[i])
		}
	}

	return pool
}

func natureSelect(popSize int, mutationRate float64, pods []*corev1.Pod, nodes []*corev1.Node, pool []individual) []individual {
	nextGeneration := make([]individual, popSize)

	for population := 0; population < popSize; {
		idv1, idv2 := pool[rand.Intn(len(pool))], pool[rand.Intn(len(pool))]
		child := crossover(idv1, idv2)
		child = mutate(child, mutationRate, len(nodes))
		fitness, flag := calculateIndividualFitness(child, pods, nodes)

		if flag == 1 {
			child.fitness = fitness / 2
		} else {
			child.fitness = fitness
		}
		child.flag = flag
		nextGeneration[population] = child
		population++

	}
	return nextGeneration
}

func crossover(idv1 individual, idv2 individual) individual {
	child := individual{
		podsCode: make([]int, 0),
	}

	mid := len(idv1.podsCode) / 2
	child.podsCode = append(child.podsCode, idv1.podsCode[0:mid]...)
	child.podsCode = append(child.podsCode, idv2.podsCode[mid:]...)

	return child
}

func mutate(idv individual, mutationRate float64, nodeNum int) individual {
	if rand.Float64() <= mutationRate {
		idv.podsCode[rand.Intn(len(idv.podsCode))] = rand.Intn(nodeNum)
	}

	return idv
}

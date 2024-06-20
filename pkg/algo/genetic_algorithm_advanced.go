package algo

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"math/rand"
)

//type individual struct {
//	podsCode []int
//	fitness  float64
//}

func initCodes() [][]int {
	codes := [][]int{
		{9, 13, 13, 5, 11, 3, 6, 9, 3, 6, 7, 1, 0, 6, 14, 5, 7, 7, 3, 6, 13, 0, 4, 5, 8, 14, 9, 4, 8, 13, 11, 5, 2, 2, 2, 4, 12, 13, 3, 10, 8, 4, 4, 1, 0, 10, 8, 7},
		{8, 10, 11, 2, 1, 9, 5, 2, 13, 9, 8, 3, 5, 9, 2, 9, 9, 7, 2, 7, 0, 4, 12, 14, 6, 13, 10, 8, 2, 12, 12, 14, 7, 7, 6, 0, 3, 8, 0, 6, 14, 12, 0, 3, 11, 13, 11, 4},
		{14, 9, 8, 2, 9, 4, 5, 2, 1, 12, 4, 13, 10, 14, 9, 3, 7, 11, 7, 7, 7, 12, 11, 5, 14, 12, 10, 0, 3, 0, 10, 1, 8, 1, 11, 5, 4, 8, 6, 13, 13, 9, 8, 13, 4, 3, 10, 2},
		{8, 7, 7, 3, 9, 0, 1, 8, 11, 5, 12, 8, 9, 12, 13, 4, 11, 0, 0, 5, 12, 3, 6, 5, 1, 13, 0, 10, 7, 4, 2, 9, 0, 7, 9, 6, 12, 2, 2, 14, 1, 8, 4, 14, 10, 14, 1, 2},
		{4, 5, 0, 7, 7, 11, 6, 14, 9, 7, 0, 0, 2, 5, 14, 6, 9, 6, 0, 11, 6, 11, 4, 14, 4, 11, 1, 3, 7, 9, 8, 12, 9, 8, 10, 3, 5, 2, 3, 12, 5, 14, 1, 12, 6, 2, 1, 8},
		{12, 8, 11, 4, 12, 8, 0, 12, 7, 8, 8, 0, 2, 11, 4, 11, 13, 14, 7, 9, 13, 12, 4, 2, 14, 0, 5, 5, 12, 6, 0, 10, 7, 13, 1, 10, 2, 4, 9, 0, 13, 5, 14, 14, 6, 11, 3, 5},
		{8, 11, 0, 5, 2, 1, 14, 6, 2, 6, 4, 5, 7, 1, 5, 8, 0, 9, 4, 5, 6, 12, 12, 12, 14, 0, 10, 14, 14, 0, 10, 13, 13, 4, 10, 8, 3, 13, 14, 3, 9, 9, 3, 1, 3, 7, 7, 4},
		{8, 8, 5, 13, 2, 9, 7, 12, 2, 6, 4, 4, 3, 6, 4, 11, 13, 0, 10, 0, 12, 14, 8, 7, 13, 4, 12, 3, 5, 6, 0, 3, 8, 11, 5, 5, 2, 14, 12, 9, 9, 0, 14, 9, 1, 13, 14, 11},
		{1, 9, 8, 4, 2, 2, 4, 7, 3, 5, 8, 0, 8, 14, 7, 14, 6, 3, 8, 0, 14, 5, 1, 7, 10, 2, 4, 14, 6, 4, 3, 5, 0, 11, 13, 13, 0, 11, 8, 7, 12, 6, 6, 2, 1, 11, 1, 9},
		{11, 1, 9, 14, 6, 13, 0, 5, 5, 6, 1, 7, 3, 9, 8, 6, 1, 3, 13, 4, 4, 3, 9, 6, 1, 13, 5, 12, 3, 11, 8, 13, 12, 3, 2, 2, 11, 4, 13, 11, 9, 2, 14, 10, 12, 2, 10, 0},
		{9, 13, 13, 2, 14, 4, 10, 12, 0, 0, 4, 6, 9, 8, 10, 5, 10, 4, 1, 3, 5, 13, 3, 1, 11, 6, 9, 2, 4, 6, 0, 7, 2, 12, 7, 8, 4, 1, 1, 8, 5, 1, 3, 7, 10, 6, 12, 14},
		{1, 7, 5, 10, 5, 5, 14, 2, 8, 2, 2, 4, 7, 7, 14, 6, 10, 4, 8, 8, 7, 1, 9, 5, 6, 14, 13, 14, 1, 8, 13, 9, 9, 3, 11, 11, 0, 12, 0, 11, 2, 6, 3, 13, 0, 1, 0, 4},
		{12, 8, 5, 10, 7, 1, 4, 5, 10, 2, 1, 11, 9, 13, 10, 12, 7, 7, 1, 4, 14, 12, 11, 11, 10, 1, 9, 5, 12, 7, 3, 9, 0, 6, 4, 2, 8, 11, 14, 9, 3, 11, 3, 3, 6, 8, 7, 14},
		{2, 4, 11, 0, 14, 10, 9, 2, 14, 6, 7, 8, 10, 4, 12, 2, 3, 9, 1, 2, 10, 11, 10, 14, 8, 5, 9, 9, 12, 3, 13, 3, 0, 6, 3, 12, 12, 8, 9, 7, 7, 10, 1, 0, 8, 6, 1, 5},
		{10, 1, 3, 0, 4, 9, 8, 5, 9, 4, 6, 6, 14, 5, 12, 10, 2, 8, 3, 12, 10, 8, 10, 7, 12, 13, 13, 10, 9, 1, 5, 13, 0, 14, 2, 11, 7, 2, 4, 8, 8, 14, 14, 2, 9, 5, 11, 0},
		{7, 7, 3, 1, 3, 4, 9, 2, 14, 10, 6, 12, 7, 1, 0, 13, 3, 10, 5, 0, 11, 12, 12, 2, 13, 3, 10, 13, 10, 4, 8, 0, 4, 14, 6, 8, 2, 8, 6, 8, 6, 3, 6, 1, 12, 9, 11, 14},
		{13, 12, 6, 3, 1, 14, 10, 9, 11, 4, 3, 11, 4, 9, 1, 6, 4, 11, 3, 13, 4, 2, 3, 9, 12, 14, 10, 0, 8, 1, 11, 5, 14, 1, 5, 6, 12, 1, 14, 10, 6, 10, 8, 8, 10, 7, 8, 2},
		{11, 10, 7, 13, 5, 12, 14, 8, 10, 8, 12, 13, 5, 2, 13, 10, 2, 13, 8, 9, 14, 4, 9, 1, 14, 2, 9, 4, 3, 5, 12, 11, 3, 14, 0, 5, 12, 0, 11, 10, 7, 4, 2, 11, 3, 3, 1, 1},
		{6, 11, 8, 4, 3, 1, 4, 7, 1, 8, 4, 13, 2, 8, 10, 12, 9, 7, 0, 0, 11, 8, 10, 13, 4, 14, 12, 13, 14, 3, 5, 12, 1, 11, 3, 7, 6, 2, 2, 1, 11, 9, 7, 10, 10, 9, 2, 12},
		{10, 3, 9, 2, 7, 7, 11, 5, 4, 9, 7, 10, 2, 10, 3, 7, 10, 5, 1, 5, 4, 9, 4, 1, 1, 11, 3, 12, 7, 5, 0, 3, 0, 2, 11, 14, 8, 2, 6, 5, 7, 0, 1, 13, 13, 13, 11, 12},
		{6, 3, 6, 10, 9, 11, 12, 8, 10, 5, 13, 9, 4, 4, 1, 4, 12, 4, 14, 11, 14, 3, 4, 1, 5, 12, 5, 7, 2, 13, 8, 13, 12, 6, 10, 2, 13, 9, 7, 9, 9, 13, 10, 3, 1, 2, 2, 11},
		{5, 6, 2, 11, 1, 3, 6, 9, 3, 3, 12, 13, 0, 4, 0, 5, 6, 8, 7, 2, 2, 6, 11, 3, 6, 12, 7, 8, 14, 10, 14, 7, 1, 13, 8, 7, 9, 4, 5, 8, 11, 9, 11, 4, 5, 13, 10, 1},
		{3, 14, 14, 6, 1, 12, 0, 3, 8, 6, 4, 10, 9, 5, 0, 0, 12, 14, 14, 11, 14, 0, 4, 10, 2, 11, 5, 8, 7, 6, 4, 7, 1, 6, 2, 12, 10, 10, 12, 2, 8, 4, 1, 7, 1, 5, 11, 13},
		{10, 14, 14, 7, 2, 4, 13, 11, 2, 14, 8, 0, 7, 2, 6, 2, 14, 5, 1, 3, 13, 4, 9, 1, 5, 10, 3, 0, 2, 12, 8, 3, 7, 1, 10, 0, 9, 4, 5, 11, 5, 6, 8, 13, 4, 11, 12, 6},
		{2, 0, 3, 6, 7, 5, 9, 1, 0, 5, 3, 2, 4, 8, 6, 5, 13, 14, 0, 2, 4, 11, 4, 9, 7, 10, 3, 8, 10, 1, 8, 10, 7, 12, 12, 7, 9, 10, 10, 11, 4, 5, 0, 12, 9, 11, 3, 6},
		{14, 11, 2, 9, 14, 7, 0, 10, 11, 3, 3, 9, 6, 0, 8, 0, 9, 10, 9, 11, 3, 0, 7, 0, 8, 8, 5, 3, 13, 10, 12, 12, 13, 6, 12, 2, 7, 9, 12, 8, 14, 6, 1, 14, 2, 1, 1, 6},
		{9, 13, 4, 3, 9, 8, 13, 6, 10, 9, 4, 13, 3, 0, 10, 12, 6, 4, 8, 7, 14, 10, 7, 9, 3, 2, 6, 13, 1, 5, 8, 0, 7, 2, 8, 14, 2, 6, 5, 0, 3, 11, 11, 0, 11, 5, 12, 11},
		{9, 8, 5, 0, 5, 12, 12, 0, 8, 11, 6, 7, 14, 12, 10, 2, 8, 8, 3, 9, 7, 2, 12, 1, 5, 14, 2, 2, 6, 14, 3, 3, 11, 3, 14, 11, 5, 4, 10, 4, 1, 4, 0, 10, 13, 13, 9, 4},
		{3, 0, 14, 11, 8, 14, 13, 8, 3, 8, 10, 6, 8, 12, 12, 9, 10, 12, 3, 10, 10, 7, 14, 7, 12, 0, 2, 14, 6, 11, 1, 2, 7, 13, 4, 0, 2, 10, 4, 12, 1, 5, 2, 6, 9, 11, 5, 13},
		{12, 9, 5, 8, 1, 11, 14, 13, 7, 2, 0, 13, 12, 6, 11, 7, 11, 7, 2, 5, 0, 0, 2, 7, 13, 6, 5, 1, 0, 1, 3, 3, 11, 2, 6, 4, 9, 10, 14, 9, 12, 5, 9, 12, 4, 8, 1, 14},
		{2, 4, 0, 4, 0, 14, 9, 10, 8, 3, 1, 12, 0, 6, 13, 3, 9, 11, 10, 1, 8, 5, 14, 8, 14, 13, 4, 7, 6, 9, 0, 13, 11, 2, 13, 7, 11, 11, 6, 10, 1, 7, 11, 10, 5, 5, 14, 3},
		{5, 5, 9, 1, 2, 0, 2, 12, 14, 5, 2, 2, 12, 13, 7, 13, 12, 7, 3, 6, 10, 12, 11, 6, 4, 2, 0, 10, 8, 13, 11, 8, 13, 1, 0, 11, 9, 13, 2, 12, 10, 11, 10, 4, 10, 7, 1, 4},
		{13, 4, 9, 7, 14, 5, 0, 1, 11, 11, 9, 14, 14, 0, 9, 14, 6, 10, 12, 13, 12, 1, 8, 3, 3, 4, 6, 0, 9, 11, 10, 11, 7, 4, 2, 1, 3, 12, 1, 12, 9, 12, 14, 7, 5, 10, 0, 6},
		{13, 4, 5, 13, 5, 6, 8, 5, 10, 14, 13, 10, 1, 0, 9, 8, 12, 12, 11, 8, 13, 4, 14, 14, 2, 3, 12, 9, 8, 13, 0, 0, 1, 9, 4, 1, 2, 7, 2, 14, 5, 8, 1, 3, 3, 10, 6, 2},
		{13, 1, 1, 4, 13, 4, 13, 5, 5, 8, 9, 10, 4, 8, 8, 11, 12, 12, 12, 13, 9, 4, 11, 3, 7, 2, 5, 8, 13, 10, 14, 2, 10, 7, 10, 0, 0, 7, 9, 2, 0, 9, 1, 9, 2, 7, 0, 6},
		{0, 14, 1, 1, 7, 9, 6, 3, 1, 9, 11, 13, 10, 13, 14, 3, 9, 11, 6, 10, 12, 12, 13, 3, 4, 13, 12, 8, 11, 5, 10, 0, 0, 10, 6, 5, 6, 8, 0, 7, 9, 9, 2, 8, 7, 5, 7, 8},
		{6, 7, 11, 11, 10, 8, 2, 6, 14, 10, 2, 11, 11, 10, 7, 2, 14, 4, 14, 12, 6, 0, 7, 14, 8, 0, 8, 1, 3, 2, 3, 10, 5, 4, 9, 8, 0, 13, 5, 5, 5, 5, 9, 3, 7, 12, 9, 4},
		{11, 6, 11, 14, 2, 9, 9, 7, 14, 0, 1, 7, 2, 2, 10, 11, 11, 6, 0, 7, 0, 6, 5, 14, 1, 13, 10, 4, 1, 0, 12, 7, 8, 9, 6, 3, 10, 3, 1, 5, 4, 0, 13, 5, 4, 13, 1, 4},
		{11, 9, 1, 3, 5, 2, 6, 12, 1, 1, 9, 7, 2, 3, 14, 9, 5, 13, 7, 11, 5, 4, 11, 5, 14, 2, 14, 2, 6, 8, 9, 13, 10, 10, 5, 8, 10, 13, 11, 14, 0, 13, 9, 4, 0, 7, 0, 12},
		{12, 3, 9, 4, 14, 12, 3, 6, 8, 3, 0, 11, 13, 8, 4, 4, 3, 11, 13, 11, 12, 14, 14, 7, 12, 0, 8, 0, 0, 9, 1, 11, 9, 1, 7, 13, 7, 3, 4, 2, 10, 6, 2, 5, 6, 1, 5, 13},
		{14, 11, 8, 13, 12, 11, 1, 4, 4, 14, 0, 0, 8, 2, 8, 7, 10, 3, 12, 12, 5, 11, 10, 4, 9, 7, 5, 1, 13, 8, 12, 8, 6, 3, 9, 6, 0, 3, 6, 11, 0, 1, 1, 5, 13, 2, 13, 9},
		{2, 11, 7, 4, 6, 6, 12, 7, 2, 8, 13, 6, 6, 7, 0, 7, 8, 9, 5, 11, 4, 14, 2, 1, 5, 8, 6, 4, 12, 12, 11, 14, 3, 1, 14, 12, 13, 5, 0, 0, 4, 9, 11, 0, 5, 13, 9, 1},
		{5, 11, 2, 4, 2, 11, 7, 5, 5, 10, 8, 10, 4, 9, 0, 0, 0, 8, 14, 0, 12, 6, 8, 14, 3, 1, 3, 14, 6, 11, 2, 6, 7, 8, 4, 11, 7, 10, 12, 4, 6, 1, 13, 13, 7, 12, 14, 13},
		{14, 1, 7, 4, 3, 5, 4, 0, 9, 5, 7, 11, 7, 9, 10, 8, 2, 13, 3, 8, 8, 7, 4, 5, 9, 0, 0, 1, 9, 14, 13, 12, 11, 12, 10, 3, 5, 3, 7, 8, 1, 11, 11, 13, 6, 6, 14, 2},
		{0, 4, 13, 11, 13, 5, 0, 14, 11, 11, 8, 10, 7, 5, 13, 14, 13, 2, 8, 14, 7, 3, 6, 8, 4, 9, 9, 12, 4, 5, 8, 2, 12, 3, 12, 6, 10, 10, 12, 1, 0, 8, 14, 9, 1, 1, 5, 3},
		{8, 6, 10, 6, 9, 7, 6, 3, 5, 14, 0, 2, 7, 12, 1, 13, 7, 12, 8, 14, 11, 9, 11, 12, 11, 0, 14, 8, 6, 4, 4, 9, 8, 10, 10, 1, 5, 13, 6, 4, 3, 7, 13, 1, 9, 12, 3, 2},
		{8, 9, 13, 1, 10, 9, 9, 9, 7, 7, 13, 10, 6, 4, 14, 2, 4, 14, 7, 8, 3, 14, 12, 5, 0, 13, 11, 6, 3, 13, 14, 7, 1, 5, 3, 11, 1, 3, 3, 12, 0, 11, 9, 6, 0, 0, 12, 11},
		{1, 10, 9, 10, 4, 0, 13, 14, 14, 13, 3, 8, 11, 14, 13, 12, 13, 5, 6, 3, 2, 8, 0, 10, 0, 3, 3, 1, 6, 6, 1, 2, 7, 5, 9, 2, 12, 4, 7, 12, 1, 10, 11, 4, 5, 7, 8, 6},
		{10, 14, 14, 11, 11, 9, 3, 14, 6, 9, 1, 0, 11, 3, 1, 9, 12, 8, 9, 0, 12, 6, 1, 11, 4, 0, 12, 1, 4, 0, 4, 13, 6, 2, 13, 7, 5, 8, 5, 1, 7, 14, 12, 13, 7, 2, 8, 10},
		{5, 2, 12, 8, 12, 3, 12, 14, 1, 7, 13, 14, 1, 9, 0, 4, 6, 8, 11, 0, 4, 14, 3, 5, 3, 14, 7, 9, 11, 0, 10, 9, 0, 1, 10, 3, 10, 8, 11, 11, 2, 9, 7, 5, 13, 11, 8, 13},
	}

	return codes
}

func GeneticAlgorithm2(popSize int, mutationRate float64, iteration int, pods []*corev1.Pod, nodes []*corev1.Node) int {
	codes := initCodes()
	//codes = [][]int{}

	organism := createOrganism2(popSize, pods, nodes, codes)
	if len(organism) == 0 {
		return -1
	}

	bestIndividual, worstIndividual := getBestAndWorstIndividual(organism)
	fmt.Printf("%.3f %.3f\n", bestIndividual.fitness, worstIndividual.fitness)

	fmt.Println("Original organism")
	for i := range organism {
		fmt.Printf("%.3f ", organism[i].fitness)
	}
	fmt.Println()

	for generation := 0; generation < iteration; generation++ {
		//pool := createPool(organism, bestIndividual.fitness)
		pool := createPool2(organism, bestIndividual.fitness, worstIndividual.fitness)

		organism = natureSelect2(popSize, mutationRate, bestIndividual, pods, nodes, pool)

		bestIndividual, worstIndividual = getBestAndWorstIndividual(organism)
		fmt.Printf("%.3f\n", bestIndividual.fitness)
	}

	fmt.Println("Final organism")
	for i := range organism {
		fmt.Printf("%.3f ", organism[i].fitness)
	}
	fmt.Println()

	for i, podCode := range bestIndividual.podsCode {
		pods[i].Spec.NodeName = nodes[podCode].Name
	}

	return 0
}

func getBestAndWorstIndividual(organism []individual) (individual, individual) {
	bestIdvNumber, worstIdvNumber := 0, 0

	for i := range organism {
		if organism[i].fitness > organism[bestIdvNumber].fitness {
			bestIdvNumber = i
		}

		if organism[i].fitness < organism[worstIdvNumber].fitness {
			worstIdvNumber = i
		}
	}

	return organism[bestIdvNumber], organism[worstIdvNumber]
}

func createOrganism2(popSize int, pods []*corev1.Pod, nodes []*corev1.Node, codes [][]int) []individual {
	organism := make([]individual, popSize)

	population := 0
	for i := range codes {
		newIndividual := individual{
			podsCode: codes[i],
		}
		fitness, _ := calculateIndividualFitness(newIndividual, pods, nodes)
		newIndividual.fitness = fitness
		organism[population] = newIndividual
		population++
	}

	podNum, nodeNum := len(pods), len(nodes)
	for population < popSize {
		newIndividual := createIndividual(podNum, nodeNum)
		fitness, flag := calculateIndividualFitness(newIndividual, pods, nodes)
		if flag == 0 {
			newIndividual.fitness = fitness
			organism[population] = newIndividual
			population++
			fmt.Println(population)
		}
	}

	return organism
}

func createPool2(organism []individual, maxFitness float64, minFitness float64) []individual {
	pool := make([]individual, 0)

	if maxFitness == minFitness {
		for i := range organism {
			for j := 0; j < 50; j++ {
				pool = append(pool, organism[i])
			}
		}
	} else {
		for i := range organism {
			num := int(25 + ((organism[i].fitness-minFitness)/(maxFitness-minFitness))*25)
			for j := 0; j < num; j++ {
				pool = append(pool, organism[i])
			}
		}
	}

	//for i := range organism {
	//	num := int((organism[i].fitness / maxFitness) * 50)
	//	for j := 0; j < num; j++ {
	//		pool = append(pool, organism[i])
	//	}
	//}

	return pool
}

func natureSelect2(popSize int, mutationRate float64, bestIdv individual, pods []*corev1.Pod, nodes []*corev1.Node, pool []individual) []individual {
	nextGeneration := make([]individual, popSize)

	nextGeneration[0] = bestIdv
	for population := 1; population < popSize; {
		idv1, idv2 := pool[rand.Intn(len(pool))], pool[rand.Intn(len(pool))]
		child := crossover2(idv1, idv2, pods, nodes)
		child = mutate2(child, mutationRate, pods, nodes)

		nextGeneration[population] = child
		population++
	}
	return nextGeneration
}

func crossover2(idv1 individual, idv2 individual, pods []*corev1.Pod, nodes []*corev1.Node) individual {
	if rand.Float64() <= 0.9 {
		child1 := individual{
			podsCode: make([]int, 0),
		}
		child2 := individual{
			podsCode: make([]int, 0),
		}

		for i := 1; i < len(idv1.podsCode); i++ {
			position := rand.Intn(len(idv1.podsCode))

			child1.podsCode, child2.podsCode = []int{}, []int{}
			child1.podsCode = append(child1.podsCode, idv1.podsCode[0:position]...)
			child1.podsCode = append(child1.podsCode, idv2.podsCode[position:]...)
			child2.podsCode = append(child2.podsCode, idv2.podsCode[0:position]...)
			child2.podsCode = append(child2.podsCode, idv1.podsCode[position:]...)

			fitness1, flag1 := calculateIndividualFitness(child1, pods, nodes)
			fitness2, flag2 := calculateIndividualFitness(child2, pods, nodes)
			if flag1 == 0 && flag2 == 0 {
				if fitness1 > fitness2 {
					child1.fitness = fitness1
					return child1
				}

				child2.fitness = fitness2
				return child2
			} else if flag1 == 0 {
				child1.fitness = fitness1
				return child1
			} else if flag2 == 0 {
				child2.fitness = fitness2
				return child2
			}
		}
	}

	if idv1.fitness > idv2.fitness {
		return idv1
	}
	return idv2
}

func mutate2(idv individual, mutationRate float64, pods []*corev1.Pod, nodes []*corev1.Node) individual {
	if rand.Float64() <= mutationRate {
		for {
			newIdv := individual{
				podsCode: make([]int, 0),
			}
			newIdv.podsCode = append(newIdv.podsCode, idv.podsCode...)
			newIdv.podsCode[rand.Intn(len(pods))] = rand.Intn(len(nodes))

			fitness, flag := calculateIndividualFitness(newIdv, pods, nodes)
			if flag == 0 {
				newIdv.fitness = fitness
				return newIdv
			}
		}
	}

	return idv
}

package simulator

import (
	"context"
	"fmt"
	"github.com/alibaba/open-simulator/pkg/algo"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"

	"github.com/pterm/pterm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	kubeinformers "k8s.io/client-go/informers"
	externalclientset "k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/scheduler"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	utiltrace "k8s.io/utils/trace"

	simonplugin "github.com/alibaba/open-simulator/pkg/simulator/plugin"
	"github.com/alibaba/open-simulator/pkg/test"
	simontype "github.com/alibaba/open-simulator/pkg/type"
	"github.com/alibaba/open-simulator/pkg/utils"
	"k8s.io/client-go/tools/events"
)

// Simulator is used to simulate a cluster and pods scheduling
type Simulator struct {
	// kube client
	kubeclient      externalclientset.Interface
	fakeclient      externalclientset.Interface
	informerFactory informers.SharedInformerFactory

	// scheduler
	scheduler *scheduler.Scheduler

	// stopCh
	simulatorStop chan struct{}

	// context
	ctx                   context.Context
	cancelFunc            context.CancelFunc
	scheduleOneCtx        context.Context
	scheduleOneCancelFunc context.CancelFunc

	eventBroadcaster events.EventBroadcasterAdapter

	disablePTerm    bool
	patchPodFuncMap PatchPodsFuncMap

	nodesInfo map[string]*framework.NodeInfo
	status    status
}

// status captures reason why one pod fails to be scheduled
type status struct {
	stopReason string
}

type PatchPodFunc = func(pods []*corev1.Pod, client externalclientset.Interface) error

type PatchPodsFuncMap map[string]PatchPodFunc

type simulatorOptions struct {
	kubeconfig      string
	schedulerConfig string
	disablePTerm    bool
	extraRegistry   frameworkruntime.Registry
	patchPodFuncMap PatchPodsFuncMap
}

// Option configures a Simulator
type Option func(*simulatorOptions)

var defaultSimulatorOptions = simulatorOptions{
	kubeconfig:      "",
	schedulerConfig: "",
	disablePTerm:    false,
	extraRegistry:   make(map[string]frameworkruntime.PluginFactory),
	patchPodFuncMap: make(map[string]PatchPodFunc),
}

// NewSimulator generates all components that will be needed to simulate scheduling and returns a complete simulator
func NewSimulator(nodesInfo map[string]*framework.NodeInfo, opts ...Option) (*Simulator, error) {
	var err error
	// Step 0: configures a Simulator by opts
	options := defaultSimulatorOptions
	for _, opt := range opts {
		opt(&options)
	}

	// Step 1: get scheduler CompletedConfig and set the list of scheduler bind plugins to Simon.
	kubeSchedulerConfig, err := GetAndSetSchedulerConfig(options.schedulerConfig)
	if err != nil {
		return nil, err
	}

	// Step 2: create client
	fakeClient := fakeclientset.NewSimpleClientset()
	kubeclient, err := utils.CreateKubeClient(options.kubeconfig)
	if err != nil {
		kubeclient = nil
	}
	kubeSchedulerConfig.Client = fakeClient

	// Step 3: Create the simulator
	ctx, cancel := context.WithCancel(context.Background())
	scheduleOneCtx, scheduleOneCancel := context.WithCancel(context.Background())
	sim := &Simulator{
		fakeclient:            fakeClient,
		kubeclient:            kubeclient,
		simulatorStop:         make(chan struct{}),
		ctx:                   ctx,
		cancelFunc:            cancel,
		scheduleOneCtx:        scheduleOneCtx,
		scheduleOneCancelFunc: scheduleOneCancel,
		disablePTerm:          options.disablePTerm,
		patchPodFuncMap:       options.patchPodFuncMap,
		eventBroadcaster:      kubeSchedulerConfig.EventBroadcaster,
		nodesInfo:             nodesInfo,
	}

	// Step 4: create informer
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(sim.fakeclient, 0)
	storagev1Informers := kubeInformerFactory.Storage().V1()
	scInformer := kubeInformerFactory.Storage().V1().StorageClasses().Informer()
	csiNodeInformer := kubeInformerFactory.Storage().V1().CSINodes().Informer()
	cmInformer := kubeInformerFactory.Core().V1().ConfigMaps().Informer()
	svcInformer := kubeInformerFactory.Core().V1().Services().Informer()
	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()
	pdbInformer := kubeInformerFactory.Policy().V1beta1().PodDisruptionBudgets().Informer()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer()
	pvInformer := kubeInformerFactory.Core().V1().PersistentVolumes().Informer()
	rcInformer := kubeInformerFactory.Core().V1().ReplicationControllers().Informer()
	rsInformer := kubeInformerFactory.Apps().V1().ReplicaSets().Informer()
	stsInformer := kubeInformerFactory.Apps().V1().StatefulSets().Informer()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes().Informer()
	dsInformer := kubeInformerFactory.Apps().V1().DaemonSets().Informer()
	deployInformer := kubeInformerFactory.Apps().V1().Deployments().Informer()

	// Step 5: add event handler for pods
	kubeInformerFactory.Core().V1().Pods().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				if pod, ok := obj.(*corev1.Pod); ok && pod.Spec.SchedulerName == simontype.DefaultSchedulerName {
					return true
				}
				return false
			},
			Handler: cache.ResourceEventHandlerFuncs{
				// AddFunc: func(obj interface{}) {
				// 	if pod, ok := obj.(*corev1.Pod); ok {
				// 		fmt.Printf("test add pod %s/%s\n", pod.Namespace, pod.Name)
				// 	}
				// },
				UpdateFunc: func(oldObj, newObj interface{}) {
					if pod, ok := newObj.(*corev1.Pod); ok {
						// fmt.Printf("test update pod %s/%s\n", pod.Namespace, pod.Name)
						sim.update(pod)
					}
				},
			},
		},
	)
	sim.informerFactory = kubeInformerFactory

	// Step 6: start informer
	sim.informerFactory.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(),
		scInformer.HasSynced,
		csiNodeInformer.HasSynced,
		cmInformer.HasSynced,
		svcInformer.HasSynced,
		podInformer.HasSynced,
		pdbInformer.HasSynced,
		pvcInformer.HasSynced,
		pvInformer.HasSynced,
		rcInformer.HasSynced,
		rsInformer.HasSynced,
		stsInformer.HasSynced,
		nodeInformer.HasSynced,
		dsInformer.HasSynced,
		deployInformer.HasSynced,
	)

	// Step 7: create scheduler for sim
	bindRegistry := frameworkruntime.Registry{
		simontype.SimonPluginName: func(configuration runtime.Object, f framework.Handle) (framework.Plugin, error) {
			return simonplugin.NewSimonPlugin(sim.fakeclient, nodesInfo, configuration, f)
		},
		simontype.LeastAllocationPluginName: func(configuration runtime.Object, f framework.Handle) (framework.Plugin, error) {
			return simonplugin.NewLeastAllocationPlugin(nodesInfo, configuration, f)
		},
		simontype.OpenLocalPluginName: func(configuration runtime.Object, f framework.Handle) (framework.Plugin, error) {
			return simonplugin.NewLocalPlugin(fakeClient, storagev1Informers, configuration, f)
		},
		simontype.OpenGpuSharePluginName: func(configuration runtime.Object, f framework.Handle) (framework.Plugin, error) {
			return simonplugin.NewGpuSharePlugin(fakeClient, configuration, f)
		},
	}
	for name, plugin := range options.extraRegistry {
		bindRegistry[name] = plugin
	}
	sim.scheduler, err = scheduler.New(
		sim.fakeclient,
		sim.informerFactory,
		GetRecorderFactory(kubeSchedulerConfig),
		sim.ctx.Done(),
		scheduler.WithProfiles(kubeSchedulerConfig.ComponentConfig.Profiles...),
		scheduler.WithAlgorithmSource(kubeSchedulerConfig.ComponentConfig.AlgorithmSource),
		scheduler.WithPercentageOfNodesToScore(kubeSchedulerConfig.ComponentConfig.PercentageOfNodesToScore),
		scheduler.WithFrameworkOutOfTreeRegistry(bindRegistry),
		scheduler.WithPodMaxBackoffSeconds(kubeSchedulerConfig.ComponentConfig.PodMaxBackoffSeconds),
		scheduler.WithPodInitialBackoffSeconds(kubeSchedulerConfig.ComponentConfig.PodInitialBackoffSeconds),
		scheduler.WithExtenders(kubeSchedulerConfig.ComponentConfig.Extenders...),
	)
	if err != nil {
		return nil, err
	}

	return sim, nil
}

// RunCluster
func (sim *Simulator) RunCluster(cluster ResourceTypes) (*SimulateResult, error) {
	// start scheduler
	sim.runScheduler()

	return sim.syncClusterResourceList(cluster)
}

func (sim *Simulator) ScheduleApp(app AppResource, newNode *corev1.Node, clusterNodes []*corev1.Node) (*SimulateResult, error) {
	// 由 AppResource 生成 Pods
	appPods, err := GenerateValidPodsFromAppResources(sim.fakeclient, app.Name, app.Resource)
	if err != nil {
		return nil, err
	}
	//affinityPriority := algo.NewAffinityQueue(appPods)
	//sort.Sort(affinityPriority)
	//tolerationPriority := algo.NewTolerationQueue(appPods)
	//sort.Sort(tolerationPriority)
	//algo.SortPodsBasedOnDiff(newNode, appPods)

	if sim.kubeclient != nil {
		for _, patchPods := range sim.patchPodFuncMap {
			if err := patchPods(appPods, sim.kubeclient); err != nil {
				return nil, err
			}
		}
	}

	for _, cm := range app.Resource.ConfigMaps {
		if _, err := sim.fakeclient.CoreV1().ConfigMaps(cm.Namespace).Create(context.Background(), cm, metav1.CreateOptions{}); err != nil {
			return nil, err
		}
	}
	for _, sc := range app.Resource.StorageClasses {
		if _, err := sim.fakeclient.StorageV1().StorageClasses().Create(context.Background(), sc, metav1.CreateOptions{}); err != nil {
			return nil, err
		}
	}
	for _, pdb := range app.Resource.PodDisruptionBudgets {
		if _, err := sim.fakeclient.PolicyV1beta1().PodDisruptionBudgets(pdb.Namespace).Create(context.Background(), pdb, metav1.CreateOptions{}); err != nil {
			return nil, err
		}
	}

	start := time.Now()
	//isFinished := algo.GeneticAlgorithm(50, 0.1, 500, appPods, clusterNodes)

	isFinished := algo.GeneticAlgorithm2(50, 0.1, 500, appPods, clusterNodes)
	t := time.Since(start)
	fmt.Println(t)

	if isFinished == -1 {
		return nil, fmt.Errorf("调度遗传算法失败")
	} else {
		fmt.Println("调度遗传算法成功")
	}

	failedPod, err := sim.schedulePods(appPods)
	if err != nil {
		return nil, err
	}

	result := &SimulateResult{
		UnscheduledPods: failedPod,
		NodeStatus:      sim.getClusterNodeStatus(),
	}

	podsCode := CodingResult(result, appPods, clusterNodes)
	fmt.Printf("{%d", podsCode[0])
	for i := 1; i < len(podsCode); i++ {
		fmt.Printf(", %d", podsCode[i])
	}
	fmt.Printf("}\n")

	return result, nil
}

func (sim *Simulator) getClusterNodeStatus() []NodeStatus {
	var nodeStatues []NodeStatus
	nodeStatusMap := make(map[string]NodeStatus)
	nodes, _ := sim.fakeclient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	allPods, _ := sim.fakeclient.CoreV1().Pods(corev1.NamespaceAll).List(context.Background(), metav1.ListOptions{})

	for _, node := range nodes.Items {
		nodeStatus := NodeStatus{}
		nodeStatus.Node = node.DeepCopy()
		nodeStatus.Pods = make([]*corev1.Pod, 0)
		nodeStatusMap[node.Name] = nodeStatus
	}

	for _, pod := range allPods.Items {
		nodeStatus := nodeStatusMap[pod.Spec.NodeName]
		nodeStatus.Pods = append(nodeStatus.Pods, pod.DeepCopy())
		nodeStatusMap[pod.Spec.NodeName] = nodeStatus
	}

	for _, node := range nodes.Items {
		status := nodeStatusMap[node.Name]
		nodeStatues = append(nodeStatues, status)
	}
	return nodeStatues
}

// runScheduler
func (sim *Simulator) runScheduler() {
	go sim.scheduler.Run(sim.scheduleOneCtx)
}

// Run starts to schedule pods
func (sim *Simulator) schedulePods(pods []*corev1.Pod) ([]UnscheduledPod, error) {
	var failedPods []UnscheduledPod
	var progressBar *pterm.ProgressbarPrinter
	if !sim.disablePTerm {
		progressBar, _ = pterm.DefaultProgressbar.WithTotal(len(pods)).Start()
		defer func() {
			_, _ = progressBar.Stop()
		}()
	}

	//recordCV := make([]float64, 0)
	for _, pod := range pods {
		//recordCV = append(recordCV, reportClusterInfos(sim.nodesInfo))
		//reportClusterInfos(sim.nodesInfo)
		if !sim.disablePTerm {
			// Update the title of the progressbar.
			progressBar.UpdateTitle(fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}
		if _, err := sim.fakeclient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("%s %s/%s: %s", simontype.CreatePodError, pod.Namespace, pod.Name, err.Error())
		}

		// we send value into sim.simulatorStop channel in update() function only,
		// update() is triggered when pod without nodename is handled.
		if pod.Spec.NodeName == "" {
			<-sim.simulatorStop
		}

		if strings.Contains(sim.status.stopReason, "failed") {
			if err := sim.fakeclient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{}); err != nil {
				return nil, fmt.Errorf("%s %s/%s: %s", simontype.DeletePodError, pod.Namespace, pod.Name, err.Error())
			}
			failedPods = append(failedPods, UnscheduledPod{
				Pod:    pod,
				Reason: sim.status.stopReason,
			})
			sim.status.stopReason = ""
		} else {
			if newPod, err := sim.fakeclient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{}); err != nil {
				return nil, fmt.Errorf("%s %s/%s: %s", simontype.GetPodError, pod.Namespace, pod.Name, err.Error())
			} else {
				for nodeName := range sim.nodesInfo {
					if nodeName == newPod.Spec.NodeName {
						sim.nodesInfo[nodeName].AddPod(newPod)
						fmt.Printf("  %s\n", nodeName)
						break
					}
				}
			}
		}

		if !sim.disablePTerm {
			progressBar.Increment()
		}
	}

	//for i := range recordCV {
	//	fmt.Printf("%.3f\n", recordCV[i])
	//}

	return failedPods, nil
}

type nodeStatus struct {
	nodeName string
	nodeInfo *framework.NodeInfo
}

func reportClusterInfos(nodesInfo map[string]*framework.NodeInfo) float64 {
	//objects := make([]nodeStatus, 0)
	//for nodeName, nodeInfo := range nodesInfo {
	//	objects = append(objects, nodeStatus{nodeName, nodeInfo})
	//}
	//sort.Slice(objects, func(i, j int) bool {
	//	if objects[i].nodeName < objects[j].nodeName {
	//		return true
	//	}
	//	return false
	//})
	//
	//clusterTable := pterm.DefaultTable.WithHasHeader()
	//var clusterTableData [][]string
	//nodeTableHeader := []string{
	//	"Node",
	//	"CPU Allocatable",
	//	"CPU Requests",
	//	"Memory Allocatable",
	//	"Memory Requests",
	//	"Bandwidth Allocatable",
	//	"Bandwidth Requests",
	//	"Disk Allocatable",
	//	"Disk Requests",
	//}
	//clusterTableData = append(clusterTableData, nodeTableHeader)
	//
	//for _, object := range objects {
	//	node := object.nodeInfo.Node()
	//	allocatable := node.Status.Allocatable
	//	reqs := object.nodeInfo.Requested.ResourceList()
	//	nodeCpuReq, nodeMemoryReq, nodeBandwidthReq, nodeDiskReq := reqs[corev1.ResourceCPU], reqs[corev1.ResourceMemory], reqs[simontype.BandwidthName], reqs[simontype.DiskName]
	//	nodeCpuReqFraction := float64(nodeCpuReq.MilliValue()) / float64(allocatable.Cpu().MilliValue()) * 100
	//	nodeMemoryReqFraction := float64(nodeMemoryReq.Value()) / float64(allocatable.Memory().Value()) * 100
	//	nodeBandwidthReqFraction := float64(nodeBandwidthReq.Value()) / float64(allocatable.Name(simontype.BandwidthName, resource.BinarySI).Value()) * 100
	//	nodeDiskReqFraction := float64(nodeDiskReq.Value()) / float64(allocatable.Name(simontype.DiskName, resource.BinarySI).Value()) * 100
	//	data := []string{
	//		node.Name,
	//		allocatable.Cpu().String(),
	//		fmt.Sprintf("%s(%.1f%%)", nodeCpuReq.String(), nodeCpuReqFraction),
	//		allocatable.Memory().String(),
	//		fmt.Sprintf("%s(%.1f%%)", nodeMemoryReq.String(), nodeMemoryReqFraction),
	//		allocatable.Name(simontype.BandwidthName, resource.BinarySI).String(),
	//		fmt.Sprintf("%s(%.1f%%)", nodeBandwidthReq.String(), nodeBandwidthReqFraction),
	//		allocatable.Name(simontype.DiskName, resource.BinarySI).String(),
	//		fmt.Sprintf("%s(%.1f%%)", nodeDiskReq.String(), nodeDiskReqFraction),
	//	}
	//
	//	clusterTableData = append(clusterTableData, data)
	//}
	//
	//if err := clusterTable.WithData(clusterTableData).Render(); err != nil {
	//	pterm.FgRed.Printf("fail to render cluster table: %s\n", err.Error())
	//	os.Exit(1)
	//}
	//pterm.FgYellow.Println()

	nodeBalanceValues := make([]float64, len(nodesInfo))
	nodeNumber := 0
	for _, nodeInfo := range nodesInfo {
		nodeBalanceValues[nodeNumber], _ = utils.CalculateBalanceValue(nodeInfo.Requested, nodeInfo.Allocatable)
		nodeNumber++
	}

	nodeBalanceValueSum := float64(0)
	for _, v := range nodeBalanceValues {
		nodeBalanceValueSum += v
	}

	clusterBalanceValue := nodeBalanceValueSum / float64(len(nodesInfo))

	return clusterBalanceValue
}

func (sim *Simulator) Close() {
	sim.scheduleOneCancelFunc()
	testpod := test.MakeFakePod("test", "test", "", "")
	_, err := sim.fakeclient.CoreV1().Pods("test").Create(context.TODO(), testpod, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("simon close with error: %s\n", err.Error())
	}
	if testpod.Spec.NodeName == "" {
		<-sim.simulatorStop
	}
	sim.cancelFunc()
	close(sim.simulatorStop)
	sim.eventBroadcaster.Shutdown()
}

func (sim *Simulator) syncClusterResourceList(resourceList ResourceTypes) (*SimulateResult, error) {
	//sync node
	for _, item := range resourceList.Nodes {
		if _, err := sim.fakeclient.CoreV1().Nodes().Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy node: %v", err)
		}
	}

	//sync pdb
	for _, item := range resourceList.PodDisruptionBudgets {
		if _, err := sim.fakeclient.PolicyV1beta1().PodDisruptionBudgets(item.Namespace).Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy PDB: %v", err)
		}
	}

	//sync svc
	for _, item := range resourceList.Services {
		if _, err := sim.fakeclient.CoreV1().Services(item.Namespace).Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy service: %v", err)
		}
	}

	//sync storage class
	for _, item := range resourceList.StorageClasses {
		if _, err := sim.fakeclient.StorageV1().StorageClasses().Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy storage class: %v", err)
		}
	}

	//sync pvc
	for _, item := range resourceList.PersistentVolumeClaims {
		if _, err := sim.fakeclient.CoreV1().PersistentVolumeClaims(item.Namespace).Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy pvc: %v", err)
		}
	}

	//sync deployment
	for _, item := range resourceList.Deployments {
		if _, err := sim.fakeclient.AppsV1().Deployments(item.Namespace).Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy deployment: %v", err)
		}
	}

	//sync rs
	for _, item := range resourceList.ReplicaSets {
		if _, err := sim.fakeclient.AppsV1().ReplicaSets(item.Namespace).Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy replica set: %v", err)
		}
	}

	//sync statefulset
	for _, item := range resourceList.StatefulSets {
		if _, err := sim.fakeclient.AppsV1().StatefulSets(item.Namespace).Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy stateful set: %v", err)
		}
	}

	//sync daemonset
	for _, item := range resourceList.DaemonSets {
		if _, err := sim.fakeclient.AppsV1().DaemonSets(item.Namespace).Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy daemon set: %v", err)
		}
	}

	// sync cm
	for _, item := range resourceList.ConfigMaps {
		if _, err := sim.fakeclient.CoreV1().ConfigMaps(item.Namespace).Create(context.TODO(), item, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("unable to copy configmap: %v", err)
		}
	}

	// sync pods
	pterm.FgYellow.Printf("sync %d pod(s) to fake cluster\n", len(resourceList.Pods))
	failedPods, err := sim.schedulePods(resourceList.Pods)
	if err != nil {
		return nil, err
	}

	return &SimulateResult{
		UnscheduledPods: failedPods,
		NodeStatus:      sim.getClusterNodeStatus(),
	}, nil
}

func (sim *Simulator) update(pod *corev1.Pod) {
	var stop bool = false
	var stopReason string
	var stopMessage string
	for _, podCondition := range pod.Status.Conditions {
		// log.Infof("podCondition %v", podCondition)
		stop = podCondition.Type == corev1.PodScheduled && podCondition.Status == corev1.ConditionFalse && podCondition.Reason == corev1.PodReasonUnschedulable
		if stop {
			stopReason = podCondition.Reason
			stopMessage = podCondition.Message
			// fmt.Printf("stop is true: %s %s\n", stopReason, stopMessage)
			break
		}
	}
	// Only for pending pods provisioned by simon
	if stop {
		sim.status.stopReason = fmt.Sprintf("failed to schedule pod (%s/%s): %s: %s", pod.Namespace, pod.Name, stopReason, stopMessage)
	}
	sim.simulatorStop <- struct{}{}
}

// WithKubeConfig sets kubeconfig for Simulator, the default value is ""
func WithKubeConfig(kubeconfig string) Option {
	return func(o *simulatorOptions) {
		o.kubeconfig = kubeconfig
	}
}

// WithSchedulerConfig sets schedulerConfig for Simulator, the default value is ""
func WithSchedulerConfig(schedulerConfig string) Option {
	return func(o *simulatorOptions) {
		o.schedulerConfig = schedulerConfig
	}
}

func WithExtraRegistry(extraRegistry frameworkruntime.Registry) Option {
	return func(o *simulatorOptions) {
		o.extraRegistry = extraRegistry
	}
}

func WithPatchPodsFuncMap(patchPodsFuncMap PatchPodsFuncMap) Option {
	return func(o *simulatorOptions) {
		o.patchPodFuncMap = patchPodsFuncMap
	}
}

func DisablePTerm(disablePTerm bool) Option {
	return func(o *simulatorOptions) {
		o.disablePTerm = disablePTerm
	}
}

// CreateClusterResourceFromClient returns a ResourceTypes struct by kube-client that connects a real cluster
func CreateClusterResourceFromClient(client externalclientset.Interface, disablePTerm bool) (ResourceTypes, error) {
	var resource ResourceTypes
	var err error
	var spinner *pterm.SpinnerPrinter
	if !disablePTerm {
		spinner, _ = pterm.DefaultSpinner.WithShowTimer().Start("get resource info from kube client")
	}

	trace := utiltrace.New("Trace CreateClusterResourceFromClient")
	defer trace.LogIfLong(100 * time.Millisecond)
	nodeItems, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return resource, fmt.Errorf("unable to list nodes: %v", err)
	}
	for _, item := range nodeItems.Items {
		newItem := item
		resource.Nodes = append(resource.Nodes, &newItem)
	}
	trace.Step("CreateClusterResourceFromClient: List Node done")

	// We will regenerate pods of all workloads in the follow-up stage.
	podItems, err := client.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{ResourceVersion: "0"})
	if err != nil {
		return resource, fmt.Errorf("unable to list pods: %v", err)
	}
	pendingPods := []*corev1.Pod{}
	for _, item := range podItems.Items {
		if !utils.OwnedByDaemonset(item.OwnerReferences) && item.DeletionTimestamp == nil {
			if item.Status.Phase == corev1.PodRunning {
				newItem := item
				resource.Pods = append(resource.Pods, &newItem)
			} else if item.Status.Phase == corev1.PodPending {
				newItem := item
				pendingPods = append(pendingPods, &newItem)
			}
		}
	}
	resource.Pods = append(resource.Pods, pendingPods...)
	trace.Step("CreateClusterResourceFromClient: List Pod done")

	pdbItems, err := client.PolicyV1beta1().PodDisruptionBudgets(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return resource, fmt.Errorf("unable to list PDBs: %v", err)
	}
	for _, item := range pdbItems.Items {
		newItem := item
		resource.PodDisruptionBudgets = append(resource.PodDisruptionBudgets, &newItem)
	}

	serviceItems, err := client.CoreV1().Services(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return resource, fmt.Errorf("unable to list services: %v", err)
	}
	for _, item := range serviceItems.Items {
		newItem := item
		resource.Services = append(resource.Services, &newItem)
	}

	storageClassesItems, err := client.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return resource, fmt.Errorf("unable to list storage classes: %v", err)
	}
	for _, item := range storageClassesItems.Items {
		newItem := item
		resource.StorageClasses = append(resource.StorageClasses, &newItem)
	}

	pvcItems, err := client.CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return resource, fmt.Errorf("unable to list pvcs: %v", err)
	}
	for _, item := range pvcItems.Items {
		newItem := item
		resource.PersistentVolumeClaims = append(resource.PersistentVolumeClaims, &newItem)
	}

	cmItems, err := client.CoreV1().ConfigMaps(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return resource, fmt.Errorf("unable to list configmaps: %v", err)
	}
	for _, item := range cmItems.Items {
		newItem := item
		resource.ConfigMaps = append(resource.ConfigMaps, &newItem)
	}

	daemonSetItems, err := client.AppsV1().DaemonSets(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return resource, fmt.Errorf("unable to list daemon sets: %v", err)
	}
	for _, item := range daemonSetItems.Items {
		newItem := item
		resource.DaemonSets = append(resource.DaemonSets, &newItem)
	}
	if !disablePTerm {
		spinner.Success("get resource info from kube client done!")
	}

	return resource, nil
}

// CreateClusterResourceFromClusterConfig return a ResourceTypes struct based on the cluster config
func CreateClusterResourceFromClusterConfig(path string) (ResourceTypes, error) {
	var resource ResourceTypes
	var content []string
	var err error

	if content, err = utils.GetYamlContentFromDirectory(path); err != nil {
		return ResourceTypes{}, fmt.Errorf("failed to get the yaml content from the cluster directory(%s): %v", path, err)
	}
	if resource, err = GetObjectFromYamlContent(content); err != nil {
		return resource, err
	}

	MatchAndSetLocalStorageAnnotationOnNode(resource.Nodes, path)

	return resource, nil
}

func NewNodeInfos(nodes []*corev1.Node) map[string]*framework.NodeInfo {
	nodeInfos := make(map[string]*framework.NodeInfo)
	for i := range nodes {
		newNodeInfo := framework.NewNodeInfo()
		err := newNodeInfo.SetNode(nodes[i])
		if err != nil {
			log.Errorf("failed to set information of node: %v\n", err)
		}
		nodeInfos[nodes[i].Name] = newNodeInfo
	}

	return nodeInfos
}

func CalculateClusterBalanceValue(nodesInfo map[string]*framework.NodeInfo) float64 {
	nodeBalanceValues := make([]float64, len(nodesInfo))
	nodeNumber := 0
	for _, nodeInfo := range nodesInfo {
		nodeBalanceValues[nodeNumber], _ = utils.CalculateBalanceValue(nodeInfo.Requested, nodeInfo.Allocatable)
		nodeNumber++
	}

	nodeBalanceValueSum := float64(0)
	for _, v := range nodeBalanceValues {
		nodeBalanceValueSum += v
	}

	clusterBalanceValue := nodeBalanceValueSum / float64(len(nodesInfo))
	return clusterBalanceValue
}

func CodingResult(result *SimulateResult, pods []*corev1.Pod, nodes []*corev1.Node) []int {
	podsCode := make([]int, len(pods))

	for _, node := range result.NodeStatus {
		nodeName := node.Node.Name
		position := -1
		for i := range nodes {
			if nodes[i].Name == nodeName {
				position = i
				break
			}
		}

		for _, podInfo := range node.Pods {
			for k, object := range pods {
				if object.Name == podInfo.Name {
					podsCode[k] = position
					break
				}
			}
		}
	}

	return podsCode
}

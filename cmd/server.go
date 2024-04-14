// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"math"
	"net"
	"os"
	"runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"github.com/shimingyah/pool"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/venilnoronha/grpc-web-istio-demo/proto"
	_ "go.uber.org/automaxprocs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	emoji "gopkg.in/kyokomi/emoji.v1"

	corev1 "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

var certFile = flag.String("cert", "/etc/secrets/certs/tls.crt", "public key ")
var keyFile = flag.String("key", "/etc/secrets/certs/tls.key", "private key")
var serverTLS = flag.Bool("tls", false, "grpc with tls certificate")

type server struct{}

func (s *server) InsertEmojis(ctx context.Context, req *proto.EmojiRequest) (*proto.EmojiResponse, error) {
	log.Printf("Client says: %s", req.InputText)
	outputText := emoji.Sprint(req.InputText)
	log.Printf("Response: %s", outputText)
	return &proto.EmojiResponse{OutputText: outputText}, nil
}

func init() {
	flag.Parse()
}

var kCfg *string

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Printf("Listening on %s\n", lis.Addr())
	log.Printf("lhb runtime.GOMAXPROCS(0): %v\n", runtime.GOMAXPROCS(-1))
	log.Printf("runtime.GOMAXPROCS(0): %v\n", runtime.GOMAXPROCS(0))
	log.Printf("runtime.NumCPU(): %v\n", runtime.NumCPU())
	log.Printf("runtime.Version(): %v\n", runtime.Version())

	opts := []grpc.ServerOption{
		grpc.InitialWindowSize(pool.InitialWindowSize),
		grpc.InitialConnWindowSize(pool.InitialConnWindowSize),
		grpc.MaxSendMsgSize(pool.MaxSendMsgSize),
		grpc.MaxRecvMsgSize(pool.MaxRecvMsgSize),
		grpc.MaxConcurrentStreams(math.MaxUint32),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    pool.KeepAliveTime,
			Timeout: pool.KeepAliveTimeout,
		}),
	}
	if *serverTLS {
		fmt.Println("start grpc server with tls configuration")
		tls, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if nil != err {
			fmt.Printf("failed to create TLS: %v\n", err)
		}
		opts = append(opts, grpc.Creds(tls))
	}
	fmt.Println(os.Hostname())
	fmt.Println(os.Getenv("HOME"))

	_, err = rest.InClusterConfig()
	if err == nil {
		fmt.Println("Running in Kubernetes environment")
	} else {
		fmt.Println("Not running in Kubernetes environment")
	}

	go getClientForPod()
	go func() {
		for {
			ReadMemStats()
			getMemInfo()
			time.Sleep(time.Second * 10)
		}
	}()
	s := grpc.NewServer(opts...)
	proto.RegisterEmojiServiceServer(s, &server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func getClientForPod() {
	scheme := k8sruntime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		fmt.Println("lhb:", err)
	}

	//cfg, err := config.GetConfig()
	//if err != nil {
	//	fmt.Fprintf(os.Stderr, "Failed to get Kubernetes client config: %v\n", err)
	//	os.Exit(1)
	//}
	//
	//// 创建客户端
	//k8sClient, err := rest.RESTClientFor(cfg)
	//if err != nil {
	//	fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
	//	os.Exit(1)
	//}

	k8sClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		fmt.Printf("Failed to create a k8s controller client, %v\n", err)
		return
	}

	namespace, _ := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	podName, _ := os.Hostname()
	fmt.Println("namespace:", string(namespace))
	fmt.Println("podName", podName)

	//gvk := schema.GroupVersionKind{
	//	Group:   "",
	//	Version: "v1",
	//	Kind:    "Pod",
	//}
	//
	////gvk.Kind = gvk.Kind + "List"
	//objList := unstructured.UnstructuredList{}
	//objList.SetGroupVersionKind(gvk)
	//ctx := context.Background()
	//for {
	//	err = k8sClient.List(ctx, &objList, client.InNamespace(namespace))
	//	if err != nil {
	//		fmt.Fprintf(os.Stderr, "Failed to get Pod %s in namespace %s: %v\n", podName, string(namespace), err)
	//      continue
	//	}
	//	if k8sClient == nil {
	//		break
	//	}
	//	fmt.Println(objList.GetKind())
	//}

	//podKey := client.ObjectKey{
	//	Name:      podName,
	//	Namespace: string(namespace),
	//}
	//
	//// 创建 Pod 对象
	//pod := &corev1.Pod{}
	//
	//// 从 Kubernetes 客户端获取 Pod 对象
	//for {
	//	err = k8sClient.Get(context.Background(), podKey, pod)
	//	if err != nil {
	//		fmt.Fprintf(os.Stderr, "Failed to get Pod %s in namespace %s: %v\n", podName, string(namespace), err)
	//      continue
	//	}
	//
	//	// 打印 Pod 中每个容器的资源限制信息
	//	for _, container := range pod.Spec.Containers {
	//		fmt.Printf("Container: %s\n", container.Name)
	//		fmt.Printf("Memory Limit: %s\n", container.Resources.Limits.Memory().String())
	//		fmt.Printf("CPU Limit: %s\n", container.Resources.Limits.Cpu().String())
	//	}
	//	time.Sleep(10 * time.Second)
	//	if k8sClient == nil {
	//		break
	//	}
	//}

	nsName := types.NamespacedName{
		Name:      podName,
		Namespace: string(namespace),
	}
	obj := unstructured.Unstructured{}
	obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	})
	for {

		err = k8sClient.Get(context.Background(), nsName, &obj)
		if err != nil {
			fmt.Printf("Failed to list trace session list, %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}
		sliceInterface, b, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
		fmt.Println(sliceInterface, b, err)
		data, _ := json.Marshal(sliceInterface)
		var c []corev1.Container
		if err = json.Unmarshal(data, &c); err != nil {
			fmt.Println("Failed to unmarshal container", err)
			time.Sleep(10 * time.Second)
		}
		for k, v := range c {
			fmt.Println(k, v.Name, v.Resources.Limits.Memory().Value())
		}
		time.Sleep(10 * time.Second)

		if k8sClient == nil {
			break
		}
	}

	//config, err := config.GetConfig()
	//config, err := rest.InClusterConfig()
	//if err != nil {
	//	fmt.Println(err, "Error getting config. Failed to start LoadMonitor")
	//	return
	//}

	//clientSet, err := kubernetes.NewForConfig(config)
	//if err != nil {
	//
	//	fmt.Println(err, "Error kubernetes.NewForConfig(). Failed to start LoadMonitor")
	//	return
	//}
	//
	metricsClient, err := versioned.NewForConfig(ctrl.GetConfigOrDie())
	//if err != nil {
	//	panic(err.Error())
	//}
	////podName := os.Getenv("HOSTNAME") // 在 Kubernetes 中，Pod 名称通常设置为 HOSTNAME 环境变量
	//podName, _ := os.Hostname()
	////namespace := os.Getenv("NAMESPACE")
	//
	////if podName == "" || namespace == "" {
	////	fmt.Println("Unable to determine Pod name and namespace.")
	////	os.Exit(1)
	////}
	//
	for {
		//	pod, err := clientSet.CoreV1().Pods("istio").Get(context.Background(), podName, metav1.GetOptions{})
		//	if err != nil {
		//		fmt.Printf("Error listing pods: %s\n", err.Error())
		//		time.Sleep(5 * time.Second)
		//		continue
		//	}
		//
		metrics, err := metricsClient.MetricsV1beta1().PodMetricses("istio").Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			time.Sleep(5 * time.Second)
			fmt.Printf("Error getting metrics for pod %s: %s\n", podName, err.Error())
			continue
		}

		for _, container := range metrics.Containers {
			fmt.Println("container name: ", container.Name)
			//		fmt.Println("container name: ", spew.Sdump(container))
			if container.Name == "server" {
				memoryUsageBytes := container.Usage["memory"]
				// limitMemoryBytes := pod.Spec.Containers[0].Resources.Limits.Memory().Value()
				// requestedMemoryBytes := pod.Spec.Containers[0].Resources.Requests.Memory().Value()
				// memoryUsagePercentage := float64(memoryUsageBytes.Value()) / float64(requestedMemoryBytes) * 100
				// fmt.Printf("pod %s limit memory %v, requested memory %v\n", pod.Name, limitMemoryBytes/1024/1024, requestedMemoryBytes/1024/1024)
				// fmt.Printf("pod %s usage memory %v\n", pod.Name, memoryUsageBytes.Value()/1024/1024)
				// fmt.Printf("Pod %s memory usage: %.2f%%\n", pod.Name, memoryUsagePercentage)
				fmt.Printf("memory usage: %d\n", memoryUsageBytes.Value())
				//
				// if memoryUsagePercentage > 80 {
				// 	fmt.Printf("Pod %s memory usage is above 80%%!\n", pod.Name)
				// 	// 这里可以添加相应的处理逻辑，比如发送警报或者调整资源配置等
				// }
			}
		}

		time.Sleep(10 * time.Second) // 每30秒检查一次
	}

}

//func getClientForPod() {
//	//var kubeconfig *string
//	//if home := homedir.HomeDir(); home != "" {
//	//	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
//	//} else {
//	//	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
//	//}
//	//
//	//config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
//	//if err != nil {
//	//	config, err = rest.InClusterConfig()
//	//	if err != nil {
//	//		fmt.Fprintf(os.Stderr, "Error building kubeconfig: %s\n", err.Error())
//	//		os.Exit(1)
//	//	}
//	//}
//	//
//	//clientSet, err := kubernetes.NewForConfig(config)
//	//if err != nil {
//	//	panic(err.Error())
//	//}
//
//	//config, err := config.GetConfig()
//	config, err := rest.InClusterConfig()
//	if err != nil {
//		fmt.Println(err, "Error getting config. Failed to start LoadMonitor")
//		return
//	}
//
//	clientSet, err := kubernetes.NewForConfig(config)
//	if err != nil {
//
//		fmt.Println(err, "Error kubernetes.NewForConfig(). Failed to start LoadMonitor")
//		return
//	}
//
//	metricsClient, err := versioned.NewForConfig(config)
//	if err != nil {
//		panic(err.Error())
//	}
//	//podName := os.Getenv("HOSTNAME") // 在 Kubernetes 中，Pod 名称通常设置为 HOSTNAME 环境变量
//	podName, _ := os.Hostname()
//	//namespace := os.Getenv("NAMESPACE")
//
//	//if podName == "" || namespace == "" {
//	//	fmt.Println("Unable to determine Pod name and namespace.")
//	//	os.Exit(1)
//	//}
//
//	for {
//		pod, err := clientSet.CoreV1().Pods("istio").Get(context.Background(), podName, metav1.GetOptions{})
//		if err != nil {
//			fmt.Printf("Error listing pods: %s\n", err.Error())
//			time.Sleep(5 * time.Second)
//			continue
//		}
//
//		metrics, err := metricsClient.MetricsV1beta1().PodMetricses("istio").Get(context.Background(), podName, metav1.GetOptions{})
//		if err != nil {
//			time.Sleep(5 * time.Second)
//			fmt.Printf("Error getting metrics for pod %s: %s\n", podName, err.Error())
//			continue
//		}
//
//		for _, container := range metrics.Containers {
//			fmt.Println("container name: ", container.Name)
//			fmt.Println("container name: ", spew.Sdump(container))
//			if container.Name == "server" {
//				memoryUsageBytes := container.Usage["memory"]
//				limitMemoryBytes := pod.Spec.Containers[0].Resources.Limits.Memory().Value()
//				requestedMemoryBytes := pod.Spec.Containers[0].Resources.Requests.Memory().Value()
//				memoryUsagePercentage := float64(memoryUsageBytes.Value()) / float64(requestedMemoryBytes) * 100
//				fmt.Printf("pod %s limit memory %v, requested memory %v\n", pod.Name, limitMemoryBytes/1024/1024, requestedMemoryBytes/1024/1024)
//				fmt.Printf("pod %s usage memory %v\n", pod.Name, memoryUsageBytes.Value()/1024/1024)
//				fmt.Printf("Pod %s memory usage: %.2f%%\n", pod.Name, memoryUsagePercentage)
//
//				if memoryUsagePercentage > 80 {
//					fmt.Printf("Pod %s memory usage is above 80%%!\n", pod.Name)
//					// 这里可以添加相应的处理逻辑，比如发送警报或者调整资源配置等
//				}
//			}
//
//		}
//
//		time.Sleep(10 * time.Second) // 每30秒检查一次
//	}
//
//}

//func getClient() {
//	var kubeconfig *string
//	if home := homedir.HomeDir(); home != "" {
//		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
//	} else {
//		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
//	}
//
//	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
//	if err != nil {
//		config, err = rest.InClusterConfig()
//		if err != nil {
//			fmt.Fprintf(os.Stderr, "Error building kubeconfig: %s\n", err.Error())
//			os.Exit(1)
//		}
//	}
//
//	clientset, err := kubernetes.NewForConfig(config)
//	if err != nil {
//		panic(err.Error())
//	}
//
//	metricsClient, err := versioned.NewForConfig(config)
//	if err != nil {
//		panic(err.Error())
//	}
//
//	for {
//		pods, err := clientset.CoreV1().Pods("istio").List(context.Background(), metav1.ListOptions{})
//		if err != nil {
//			fmt.Printf("Error listing pods: %s\n", err.Error())
//			time.Sleep(5 * time.Second)
//			continue
//		}
//
//		for _, pod := range pods.Items {
//			metrics, err := metricsClient.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
//			if err != nil {
//				fmt.Printf("Error getting metrics for pod %s: %s\n", pod.Name, err.Error())
//				continue
//			}
//
//			for _, container := range metrics.Containers {
//				memoryUsageBytes := container.Usage["memory"]
//				limitMemoryBytes := pod.Spec.Containers[0].Resources.Limits.Memory().Value()
//				requestedMemoryBytes := pod.Spec.Containers[0].Resources.Requests.Memory().Value()
//				memoryUsagePercentage := float64(memoryUsageBytes.Value()) / float64(requestedMemoryBytes) * 100
//				fmt.Printf("pod %s limit memory %v, requested memory %v\n", pod.Name, limitMemoryBytes/1024/1024, requestedMemoryBytes/1024/1024)
//				fmt.Printf("pod %s usage memory %v\n", pod.Name, memoryUsageBytes.Value()/1024/1024)
//				fmt.Printf("Pod %s memory usage: %.2f%%\n", pod.Name, memoryUsagePercentage)
//
//				if memoryUsagePercentage > 80 {
//					fmt.Printf("Pod %s memory usage is above 80%%!\n", pod.Name)
//					// 这里可以添加相应的处理逻辑，比如发送警报或者调整资源配置等
//				}
//			}
//		}
//
//		time.Sleep(10 * time.Second) // 每30秒检查一次
//	}
//
//}

func k8sClients() {
	//k8sClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: rt.NewScheme()})
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(k8sClient)
}

//func getMetricsClient() {
//	cfg, err := config.GetConfig()
//	if err != nil {
//		fmt.Println(err, "Error getting config. Failed to start LoadMonitor")
//		return
//	}
//
//	clientSet, err := kubernetes.NewForConfig(cfg)
//	if err != nil {
//
//		fmt.Println(err, "Error kubernetes.NewForConfig(). Failed to start LoadMonitor")
//		return
//	}
//	fmt.Println(clientSet)
//	metricsClient, err := metricsclientset.NewForConfig(cfg)
//	if err != nil {
//		return
//	}
//	fmt.Println(metricsClient)
//}
//
//func getMetricsFromMetricsAPI(ctx context.Context, metricsClient metricsclientset.Interface, namespace, resourceName string, allNamespaces bool, selector labels.Selector) (*metricsapi.PodMetricsList, error) {
//	var err error
//	ns := metav1.NamespaceAll
//	if !allNamespaces {
//		ns = namespace
//	}
//	versionedMetrics := &metricsv1beta1api.PodMetricsList{}
//	if resourceName != "" {
//		//m, err := metricsClient.MetricsV1beta1().PodMetricses(ns).Get(context.TODO(), resourceName, metav1.GetOptions{})
//		m, err := metricsClient.MetricsV1beta1().PodMetricses(ns).Get(ctx, resourceName, metav1.GetOptions{})
//		if err != nil {
//			return nil, err
//		}
//		versionedMetrics.Items = []metricsv1beta1api.PodMetrics{*m}
//	} else {
//		//versionedMetrics, err = metricsClient.MetricsV1beta1().PodMetricses(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
//		versionedMetrics, err = metricsClient.MetricsV1beta1().PodMetricses(ns).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
//		if err != nil {
//			return nil, err
//		}
//	}
//	metrics := &metricsapi.PodMetricsList{}
//	err = metricsv1beta1api.Convert_v1beta1_PodMetricsList_To_metrics_PodMetricsList(versionedMetrics, metrics, nil)
//	if err != nil {
//		return nil, err
//	}
//	return metrics, nil
//}

func ReadMemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Println("Alloc:", m.Alloc/1024/1024)
	fmt.Println("TotalAlloc:", m.TotalAlloc/1024/1024)
	fmt.Println("Frees:", m.Frees)
	// 计算已使用内存的比例
	usedPercent := float64(m.Alloc) / float64(m.TotalAlloc) * 100

	// 判断是否超过80%
	if usedPercent >= 80 {
		fmt.Printf("Memory usage is %.2f%%, exceeds 80%%\n", usedPercent)
	} else {
		fmt.Printf("Memory usage is %.2f%%\n", usedPercent)
	}
}

func getMemInfo() {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("get memory info fail. err： ", err)
	}
	// 获取总内存大小，单位GB
	memTotal := memInfo.Total / 1024 / 1024 / 1024
	// 获取已用内存大小，单位MB
	memUsed := memInfo.Used / 1024 / 1024
	// 可用内存大小
	memAva := memInfo.Available / 1024 / 1024
	// 内存可用率
	memUsedPercent := memInfo.UsedPercent
	fmt.Printf("总内存: %v GB, 已用内存: %v MB, 可用内存: %v MB, 内存使用率: %.3f %% \n", memTotal, memUsed, memAva, memUsedPercent)
}

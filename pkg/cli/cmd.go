package cli

import (
	"context"
	"os"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/config"
	ovpa "github.com/SocialGouv/oblik/pkg/vpa"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/klog/v2"
)

func NewCommand() *cobra.Command {
	var selector string
	var name string
	var namespace string
	var all bool
	var Command = &cobra.Command{
		Use:   "cli",
		Short: "Oblik CLI",
		Run: func(cmd *cobra.Command, args []string) {
			if err := Run(namespace, name, selector, all); err != nil {
				os.Exit(1)
			}
		},
	}
	flags := Command.PersistentFlags()
	flags.StringVarP(&selector, "selector", "l", "", "Label selector for filtering VPAs")
	flags.StringVarP(&name, "name", "", "", "Name of the VPA")
	flags.StringVarP(&namespace, "namespace", "n", "", "Namespace containing VPAs")
	flags.BoolVarP(&all, "all", "a", false, "Process all namespaces")
	return Command
}

func Run(namespace string, resourceName string, selector string, all bool) error {
	// Validate input parameters
	if resourceName != "" && namespace == "" {
		klog.Fatalf("Namespace must be specified when name is provided")
		return nil
	}

	if !all && namespace == "" && resourceName == "" && selector == "" {
		klog.Fatalf("Specify at least one of namespace, selector, or use --all flag. Name requires namespace.")
		return nil
	}

	kubeClients := client.NewKubeClients()

	if resourceName != "" {
		vpaResource := getVPA(kubeClients.VpaClientset, namespace, resourceName)
		if vpaResource != nil {
			if err := processVPA(kubeClients, vpaResource); err != nil {
				return err
			}
		}
	} else {
		var vpaResources []vpa.VerticalPodAutoscaler
		if all {
			vpaResources = listAllVPAs(kubeClients.VpaClientset, selector)
		} else {
			vpaResources = listVPAs(kubeClients.VpaClientset, namespace, selector)
		}
		for _, vpaResource := range vpaResources {
			if err := processVPA(kubeClients, &vpaResource); err != nil {
				return err
			}
		}
	}
	return nil
}

func getVPA(vpaClient *vpaclientset.Clientset, namespace, name string) *vpa.VerticalPodAutoscaler {
	vpaResource, err := vpaClient.AutoscalingV1().VerticalPodAutoscalers(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		klog.Fatalf("Unable to get VPA: %v\n", err)
		return nil
	}
	klog.Infof("Found VPA: %s\n", vpaResource.Name)
	return vpaResource
}

func listVPAs(vpaClient *vpaclientset.Clientset, namespace, selector string) []vpa.VerticalPodAutoscaler {
	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}

	vpaList, err := vpaClient.AutoscalingV1().VerticalPodAutoscalers(namespace).List(context.Background(), listOptions)
	if err != nil {
		klog.Fatalf("Unable to list VPAs: %v\n", err)
		return nil
	}

	klog.Infof("Found %d VPA(s)", len(vpaList.Items))
	return vpaList.Items
}

func listAllVPAs(vpaClient *vpaclientset.Clientset, selector string) []vpa.VerticalPodAutoscaler {
	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}

	vpaList, err := vpaClient.AutoscalingV1().VerticalPodAutoscalers(metav1.NamespaceAll).List(context.Background(), listOptions)
	if err != nil {
		klog.Fatalf("Unable to list VPAs: %v\n", err)
		return nil
	}

	klog.Infof("Found %d VPA(s)", len(vpaList.Items))
	return vpaList.Items
}

func processVPA(kubeClients *client.KubeClients, vpaResource *vpa.VerticalPodAutoscaler) error {
	klog.Infof("Processing VPA: %s/%s\n", vpaResource.Namespace, vpaResource.Name)
	configurable := config.CreateConfigurable(vpaResource)
	scfg := config.CreateStrategyConfig(configurable)
	return ovpa.ApplyVPARecommendations(kubeClients.Clientset, kubeClients.DynamicClient, vpaResource, scfg)
}

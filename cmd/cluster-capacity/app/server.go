package app

import (
	"fmt"

	"log"
	"time"

	"github.com/ingvagabund/cluster-capacity/cmd/cluster-capacity/app/options"
	"github.com/ingvagabund/cluster-capacity/pkg/apiserver"
	"github.com/ingvagabund/cluster-capacity/pkg/apiserver/cache"
	"github.com/ingvagabund/cluster-capacity/pkg/framework"
	"github.com/renstrom/dedent"
	"github.com/spf13/cobra"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	_ "k8s.io/kubernetes/plugin/pkg/scheduler/algorithmprovider"
)

var (
	clusterCapacityLong = dedent.Dedent(`
		Cluster-capacity simulates API server with initial state copied from kubernetes enviroment running
		on address MASTER with its configuration specified in KUBECONFIG. Simulated API server tries to schedule number of
		pods specified by --maxLimits flag. If the --maxLimits flag is not specified, pods are scheduled till
		the simulated API server runs out of resources.
	`)

	MAXREPORTS = 100
)

func NewClusterCapacityCommand() *cobra.Command {
	opt := options.NewClusterCapacityOptions()
	cmd := &cobra.Command{
		Use:   "cluster-capacity --master MASTER --kubeconfig KUBECONFIG --podspec PODSPEC",
		Short: "Cluster-capacity is used for emulating scheduling of one or multiple pods",
		Long:  clusterCapacityLong,
		Run: func(cmd *cobra.Command, args []string) {
			err := Validate(opt)
			if err != nil {
				fmt.Println(err)
				cmd.Help()
				return
			}
			err = Run(opt)
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	opt.AddFlags(cmd.Flags())
	return cmd
}

func Validate(opt *options.ClusterCapacityOptions) error {
	if len(opt.PodSpecFile) == 0 {
		return fmt.Errorf("Pod spec file is missing")
	}

	if opt.ResourceSpaceMode != "" && opt.ResourceSpaceMode != "ResourceSpaceFull" && opt.ResourceSpaceMode != "ResourceSpacePartial" {
		return fmt.Errorf("Resource space mode not recognized. Valid values are: ResourceSpaceFull, ResourceSpacePartial")
	}

	return nil
}

func Run(opt *options.ClusterCapacityOptions) error {
	conf := options.NewClusterCapacityConfig(opt)
	err := conf.ParseAPISpec()
	if err != nil {
		return fmt.Errorf("Failed to parse pod spec file: %v ", err)
	}

	err = conf.SetDefaultScheduler()
	if err != nil {
		return fmt.Errorf("Failed to set default scheduler config: %v ", err)
	}
	err = conf.ParseAdditionalSchedulerConfigs()
	if err != nil {
		return fmt.Errorf("Failed to parse config file: %v ", err)
	}

	// only of the apiserver config file is set
	err = conf.ParseApiServerConfig()
	if err != nil {
		return fmt.Errorf("Failed to parse apiserver config file: %v ", err)
	}

	conf.KubeClient, err = getKubeClient(conf.Options.Master, conf.Options.Kubeconfig)

	if err != nil {
		return err
	}

	if opt.Period == 0 {
		_, err = runSimulator(conf)
		return err
	}

	conf.Reports = cache.NewCache(MAXREPORTS)

	watch := make(chan *framework.Report)
	go func() {
		r := apiserver.NewResource(watch, conf)
		log.Fatal(apiserver.ListenAndServe(r))
	}()

	for {
		report, err := runSimulator(conf)
		if err != nil {
			return err
		}
		watch <- report
		time.Sleep(time.Duration(opt.Period) * time.Second)
	}
}

func getKubeClient(master string, config string) (clientset.Interface, error) {
	var cfg *restclient.Config
	var err error
	if master != "" && config != "" {
		cfg, err = clientcmd.BuildConfigFromFlags(master, config)
		if err != nil {
			return nil, fmt.Errorf("unable to build config from flags: %v", err)
		}
	} else {
		cfg, err = restclient.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	kubeClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("Invalid API configuration: %v", err)
	}

	if _, err = kubeClient.Discovery().ServerVersion(); err != nil {
		return nil, fmt.Errorf("Unable to get server version: %v\n", err)
	}

	return kubeClient, nil
}

func runSimulator(s *options.ClusterCapacityConfig) (*framework.Report, error) {
	mode, err := framework.StringToResourceSpaceMode(s.Options.ResourceSpaceMode)
	if err != nil {
		return nil, err
	}

	cc, err := framework.New(s.DefaultScheduler, s.Pod, s.Options.MaxLimit, mode, s.ApiServerOptions)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(s.Schedulers); i++ {
		if err = cc.AddScheduler(s.Schedulers[i]); err != nil {
			return nil, err
		}
	}

	err = cc.SyncWithClient(s.KubeClient)
	if err != nil {
		return nil, err
	}

	err = cc.Run()
	if err != nil {
		return nil, err
	}

	// report the updated pod if the simulator is run only once
	report := cc.Report(s.Options.Period == 0)

	if s.Options.Period == 0 {
		err := report.Print(s.Options.Verbose, s.Options.OutputFormat)
		if err != nil {
			return nil, fmt.Errorf("Error while printing: %v", err)
		}
	} else {
		s.Reports.Add(report)
	}

	return report, nil
}

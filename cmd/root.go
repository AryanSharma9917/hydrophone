package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/hydrophone/pkg/client"
	"sigs.k8s.io/hydrophone/pkg/common"
	"sigs.k8s.io/hydrophone/pkg/service"
)

var (
	cfgFile          string
	kubeconfig       string
	parallel         int
	verbosity        int
	outputDir        string
	cleanup          bool
	listImages       bool
	conformance      bool
	focus            string
	skip             string
	conformanceImage string
	busyboxImage     string
	dryRun           bool
	testRepoList     string
	testRepo         string
)

var rootCmd = &cobra.Command{
	Use:   "hydrohpone",
	Short: "Hydrophone is a lightweight runner for kubernetes tests.",
	Long:  `Hydrophone is a lightweight runner for kubernetes tests.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if conformance && focus != "" {
			conformance = false
			focus = ""
			log.Fatal("specify either --conformance or --focus arguments, not both")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		client := client.NewClient()
		config, clientSet := service.Init(viper.GetString("kubeconfig"))
		client.ClientSet = clientSet
		common.PrintInfo(client.ClientSet, config)
		if cleanup {
			service.Cleanup(client.ClientSet)
		} else if listImages {
			service.PrintListImages(client.ClientSet)
		} else {
			common.ValidateArgs(client.ClientSet, config)

			service.RunE2E(client.ClientSet)
			client.PrintE2ELogs()
			client.FetchFiles(config, clientSet, viper.GetString("output-dir"))
			client.FetchExitCode()
			service.Cleanup(client.ClientSet)
		}
		log.Println("Exiting with code: ", client.ExitCode)
		os.Exit(client.ExitCode)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("Default config file (%s/hydrophone/hydrophone.yaml)", xdg.ConfigHome))
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to the kubeconfig file.")

	rootCmd.PersistentFlags().IntVar(&parallel, "parallel", 1, "number of parallel threads in test framework.")
	viper.BindPFlag("parallel", rootCmd.PersistentFlags().Lookup("parallel"))

	rootCmd.PersistentFlags().IntVar(&verbosity, "verbosity", 4, "verbosity of test framework.")
	viper.BindPFlag("verbosity", rootCmd.PersistentFlags().Lookup("verbosity"))

	rootCmd.PersistentFlags().StringVar(&outputDir, "output-dir", workingDir, "directory for logs.")
	viper.BindPFlag("output-dir", rootCmd.PersistentFlags().Lookup("output-dir"))

	rootCmd.Flags().BoolVar(&cleanup, "cleanup", false, "cleanup resources (pods, namespaces etc).")

	rootCmd.Flags().BoolVar(&listImages, "list-images", false, "list all images that will be used during conformance tests.")

	rootCmd.Flags().BoolVar(&conformance, "conformance", false, "run conformance tests.")

	rootCmd.Flags().StringVar(&focus, "focus", "", "focus runs a specific e2e test. e.g. - sig-auth. allows regular expressions.")

	rootCmd.Flags().StringVar(&skip, "skip", "", "skip specific tests. allows regular expressions.")
	viper.BindPFlag("skip", rootCmd.Flags().Lookup("skip"))

	rootCmd.Flags().StringVar(&conformanceImage, "conformance-image", "", "specify a conformance container image of your choice.")
	viper.BindPFlag("conformance-image", rootCmd.Flags().Lookup("conformance-image"))

	rootCmd.Flags().StringVar(&busyboxImage, "busybox-image", "", "specify an alternate busybox container image.")
	viper.BindPFlag("busybox-image", rootCmd.Flags().Lookup("busybox-image"))

	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "run in dry run mode.")
	viper.BindPFlag("dry-run", rootCmd.Flags().Lookup("dry-run"))

	rootCmd.Flags().StringVar(&testRepoList, "test-repo-list", "", "yaml file to override registries for test images.")
	viper.BindPFlag("test-repo-list", rootCmd.Flags().Lookup("test-repo-list"))

	rootCmd.Flags().StringVar(&testRepo, "test-repo", "", "skip specific tests. allows regular expressions.")
	viper.BindPFlag("test-repo", rootCmd.Flags().Lookup("test-repo"))
}

func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// the config will belocated under `~/.config/hydrophone.yaml` on linux
		configDir := xdg.ConfigHome
		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("hydrophone")

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				err := viper.SafeWriteConfig()
				if err != nil {
					fmt.Println("Error:", err)
				}
			} else {
				fmt.Println(err)
			}
		}
	}
	kubeconfig = service.GetKubeConfig(kubeconfig)
	viper.Set("kubeconfig", kubeconfig)
}

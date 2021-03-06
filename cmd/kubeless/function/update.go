/*
Copyright (c) 2016-2017 Bitnami

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package function

import (
	"io/ioutil"
	"strings"

	"github.com/kubeless/kubeless/pkg/langruntime"
	"github.com/kubeless/kubeless/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var updateCmd = &cobra.Command{
	Use:   "update <function_name> FLAG",
	Short: "update a function on Kubeless",
	Long:  `update a function on Kubeless`,
	Run: func(cmd *cobra.Command, args []string) {
		cli := utils.GetClientOutOfCluster()
		apiExtensionsClientset := utils.GetAPIExtensionsClientOutOfCluster()
		configLocation, err := utils.GetConfigLocation(apiExtensionsClientset)
		if err != nil {
			logrus.Fatalf("Error while fetching config location: %v", err)
		}
		controllerNamespace := configLocation.Namespace
		kubelessConfig := configLocation.Name
		config, err := cli.CoreV1().ConfigMaps(controllerNamespace).Get(kubelessConfig, metav1.GetOptions{})
		if err != nil {
			logrus.Fatalf("Unable to read the configmap: %v", err)
		}

		var lr = langruntime.New(config)
		lr.ReadConfigMap()

		if len(args) != 1 {
			logrus.Fatal("Need exactly one argument - function name")
		}
		funcName := args[0]

		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			logrus.Fatal(err)
		}
		if ns == "" {
			ns = utils.GetDefaultNamespace()
		}

		handler, err := cmd.Flags().GetString("handler")
		if err != nil {
			logrus.Fatal(err)
		}

		file, err := cmd.Flags().GetString("from-file")
		if err != nil {
			logrus.Fatal(err)
		}

		secrets, err := cmd.Flags().GetStringSlice("secrets")
		if err != nil {
			logrus.Fatal(err)
		}

		runtime, err := cmd.Flags().GetString("runtime")
		if err != nil {
			logrus.Fatal(err)
		}

		if runtime != "" && !lr.IsValidRuntime(runtime) {
			logrus.Fatalf("Invalid runtime: %s. Supported runtimes are: %s",
				runtime, strings.Join(lr.GetRuntimes(), ", "))
		}

		labels, err := cmd.Flags().GetStringSlice("label")
		if err != nil {
			logrus.Fatal(err)
		}

		envs, err := cmd.Flags().GetStringArray("env")
		if err != nil {
			logrus.Fatal(err)
		}
		runtimeImage, err := cmd.Flags().GetString("runtime-image")
		if err != nil {
			logrus.Fatal(err)
		}

		mem, err := cmd.Flags().GetString("memory")
		if err != nil {
			logrus.Fatal(err)
		}

		cpu, err := cmd.Flags().GetString("cpu")
		if err != nil {
			logrus.Fatal(err)
		}

		timeout, err := cmd.Flags().GetString("timeout")
		if err != nil {
			logrus.Fatal(err)
		}

		deps, err := cmd.Flags().GetString("dependencies")
		if err != nil {
			logrus.Fatal(err)
		}
		funcDeps := ""
		if deps != "" {
			bytes, err := ioutil.ReadFile(deps)
			if err != nil {
				logrus.Fatalf("Unable to read file %s: %v", deps, err)
			}
			funcDeps = string(bytes)
		}
		headless, err := cmd.Flags().GetBool("headless")
		if err != nil {
			logrus.Fatal(err)
		}
		port, err := cmd.Flags().GetInt32("port")
		if err != nil {
			logrus.Fatal(err)
		}
		if port <= 0 || port > 65535 {
			logrus.Fatalf("Invalid port number %d specified", port)
		}

		previousFunction, err := utils.GetFunction(funcName, ns)
		if err != nil {
			logrus.Fatal(err)
		}

		f, err := getFunctionDescription(cli, funcName, ns, handler, file, funcDeps, runtime, runtimeImage, mem, cpu, timeout, port, headless, envs, labels, secrets, previousFunction)
		if err != nil {
			logrus.Fatal(err)
		}

		kubelessClient, err := utils.GetKubelessClientOutCluster()
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Redeploying function...")
		err = utils.PatchFunctionCustomResource(kubelessClient, f)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Function %s submitted for deployment", funcName)
		logrus.Infof("Check the deployment status executing 'kubeless function ls %s'", funcName)
	},
}

func init() {
	updateCmd.Flags().StringP("runtime", "", "", "Specify runtime")
	updateCmd.Flags().StringP("handler", "", "", "Specify handler")
	updateCmd.Flags().StringP("from-file", "", "", "Specify code file")
	updateCmd.Flags().StringP("memory", "", "", "Request amount of memory for the function")
	updateCmd.Flags().StringP("cpu", "", "", "Request amount of cpu for the function.")
	updateCmd.Flags().StringSliceP("label", "", []string{}, "Specify labels of the function")
	updateCmd.Flags().StringSliceP("secrets", "", []string{}, "Specify Secrets to be mounted to the functions container. For example: --secrets mySecret")
	updateCmd.Flags().StringArrayP("env", "", []string{}, "Specify environment variable of the function")
	updateCmd.Flags().StringP("namespace", "", "", "Specify namespace for the function")
	updateCmd.Flags().StringP("dependencies", "", "", "Specify a file containing list of dependencies for the function")
	updateCmd.Flags().StringP("runtime-image", "", "", "Custom runtime image")
	updateCmd.Flags().StringP("timeout", "", "180", "Maximum timeout (in seconds) for the function to complete its execution")
	updateCmd.Flags().Bool("headless", false, "Deploy http-based function without a single service IP and load balancing support from Kubernetes. See: https://kubernetes.io/docs/concepts/services-networking/service/#headless-services")
	updateCmd.Flags().Int32("port", 8080, "Deploy http-based function with a custom port")
}

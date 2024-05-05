package main

import (
	"fmt"
	etcd "go.etcd.io/etcd/server/v3/embed"
	"k8s.io/component-base/cli"
	apiserver "k8s.io/kubernetes/cmd/kube-apiserver/app"
	controller "k8s.io/kubernetes/cmd/kube-controller-manager/app"
	scheduler "k8s.io/kubernetes/cmd/kube-scheduler/app"
	"os"
	"strings"
	"time"
)

type launchErr struct {
	exitCode int
	module   string
}

func main() {
	errChan := make(chan launchErr)
	args := os.Args

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") && strings.HasSuffix(arg, "--") {
			break
		}
		if arg == "--help" || arg == "-help" || arg == "help" || arg == "-h" {
			printHelpMsg()
			return
		}
	}

	launchedModuleCnt := 0

	if newArgs, ok := rebuildArgs(args, "etcd"); ok {
		launchedModuleCnt += 1
		go func() {
			code := runEtcd(newArgs)
			errChan <- launchErr{
				exitCode: code,
				module:   "etcd",
			}
		}()
		exitOrWait(errChan)
	}

	if newArgs, ok := rebuildArgs(args, "kube-apiserver"); ok {
		launchedModuleCnt += 1
		go func() {
			apiserverCommand := apiserver.NewAPIServerCommand()
			apiserverCommand.SetArgs(newArgs)
			code := cli.Run(apiserverCommand)
			errChan <- launchErr{
				exitCode: code,
				module:   "kube-apiserver",
			}
		}()
		exitOrWait(errChan)
	}

	if newArgs, ok := rebuildArgs(args, "kube-scheduler"); ok {
		launchedModuleCnt += 1
		go func() {
			schedulerCommand := scheduler.NewSchedulerCommand()
			schedulerCommand.SetArgs(newArgs)
			code := cli.Run(schedulerCommand)
			errChan <- launchErr{
				exitCode: code,
				module:   "kube-scheduler",
			}
		}()
		exitOrWait(errChan)
	}

	if newArgs, ok := rebuildArgs(args, "kube-controller-manager"); ok {
		launchedModuleCnt += 1
		go func() {
			controllerCommand := controller.NewControllerManagerCommand()
			controllerCommand.SetArgs(newArgs)
			code := cli.Run(controllerCommand)
			errChan <- launchErr{
				exitCode: code,
				module:   "kube-controller-manager",
			}
		}()
		exitOrWait(errChan)
	}

	if launchedModuleCnt == 0 {
		printHelpMsg()
		return
	}

	err := <-errChan
	fmt.Printf("%s exits with code %d\n", err.module, err.exitCode)
	os.Exit(err.exitCode)
}

func printHelpMsg() {
	fmt.Println("Usage:")
	fmt.Println("    apiserver --help|-h|help|-help")
	fmt.Println("    apiserver \\")
	fmt.Println("        --kube-apiserver-- $args_for_kube_apiserver \\")
	fmt.Println("        --kube-scheduler-- $args_for_kube_scheduler \\")
	fmt.Println("        --kube-controller-manager-- $args_for_kube_controller_manager \\")
	fmt.Println("        --etcd-- $args_for_etcd")
}

func rebuildArgs(args []string, module string) ([]string, bool) {
	moduleBegin := -1 // inclusive
	moduleEnd := -1   // exclusive
	for i, arg := range args {
		if arg == "--"+module+"--" {
			moduleBegin = i + 1
			break
		}
	}
	if moduleBegin == -1 {
		return nil, false
	}
	for i := moduleBegin; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") && strings.HasSuffix(arg, "--") {
			moduleEnd = i
			break
		}
	}
	if moduleEnd == -1 {
		moduleEnd = len(args)
	}
	ret := make([]string, moduleEnd-moduleBegin)
	copy(ret, args[moduleBegin:moduleEnd])

	fmt.Println("=====================================")
	fmt.Println("launching " + module)
	fmt.Printf("args rebuilt to: %+v\n", ret)
	fmt.Println("=====================================")
	return ret, true
}

func runEtcd(args []string) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "-help" || arg == "help" {
			fmt.Println("etcd --config-file {...}")
			return 0
		}
	}
	var configFile string
	for i := 0; i < len(args); i += 1 {
		arg := args[i]
		var nxt *string = nil
		if i < len(args)-1 {
			nxt = &args[i+1]
		}
		if arg == "--config-file" {
			if nxt == nil {
				fmt.Println("missing config file")
				return 1
			}
			configFile = strings.TrimSpace(*nxt)
		} else if strings.HasPrefix(arg, "--config-file=") {
			configFile = strings.TrimSpace(arg[len("--config-file="):])
		} else {
			fmt.Println("unknown argument: " + arg)
			return 1
		}
	}
	if configFile == "" {
		fmt.Println("missing --config-file")
		return 1
	}

	cfg, err := etcd.ConfigFromFile(configFile)
	if err != nil {
		fmt.Printf("Failed to read etcd config: %v\n", err)
		return 1
	}
	e, err := etcd.StartEtcd(cfg)
	if err != nil {
		fmt.Printf("Failed to start etcd: %v\n", err)
		return 1
	}
	defer e.Close()
	chnl := make(chan int)
	<-chnl
	return 1
}

func exitOrWait(errChan chan launchErr) {
	select {
	case err := <-errChan:
		fmt.Printf("%s exits with code %d\n", err.module, err.exitCode)
		os.Exit(err.exitCode)
	case <-time.After(1 * time.Second):
		break
	}
}

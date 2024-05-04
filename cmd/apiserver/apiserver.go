package main

import (
	"fmt"
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
	fmt.Println("        --kube-controller-manager-- $args_for_kube_controller_manager")
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

func exitOrWait(errChan chan launchErr) {
	select {
	case err := <-errChan:
		fmt.Printf("%s exits with code %d\n", err.module, err.exitCode)
		os.Exit(err.exitCode)
	case <-time.After(1 * time.Second):
		break
	}
}

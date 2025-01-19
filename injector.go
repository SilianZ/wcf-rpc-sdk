// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/1/15 下午11:04:00
// @Desc 注入器
package wcf_rpc_sdk

import (
	"context"
	"fmt"
	"github.com/Clov614/logging"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const libSdk = "sdk.dll"
const funcInject = "WxInitSDK"
const funcDestroy = "WxDestroySDK"

var gblDll *syscall.DLL

func log(args ...interface{}) {
	fmt.Println("\033[1;7;32m[Inj]\033[0m", time.Now().Format("20060102_150405"), args)
}

///** 初始化. 加载动态库 */
//func init() {
//	log("Load dll:", libSdk)
//	var err error
//	gblDll, err = syscall.LoadDLL(libSdk)
//	if err != nil {
//		panic(err)
//	}
//}

/** 调用库接口 */
func callFunc(funName string, title string, debug bool, port int) {
	logging.Info(title)
	// log("Find function:", fun_name, "in dll:", gbl_dll)
	fun, err := gblDll.FindProc(funName)
	if err != nil {
		panic(err)
	}

	// log("Call function:", fun)
	dbgUintptr := uintptr(0)
	if debug {
		dbgUintptr = uintptr(1)
	}
	ret, _e, errno := syscall.Syscall(fun.Addr(), dbgUintptr, 0, uintptr(port), 0)
	if ret != 0 {
		panic("Function " + fmt.Sprint(fun) + " run failed! return:" +
			fmt.Sprint(ret) + ", err:" + fmt.Sprint(_e) + ", errno:" + fmt.Sprint(errno))
	}
}

/** 监听并等待SIGINT信号 */
func waitingSignal(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	logging.Info("Is running, press Ctrl+C to quit.")
	select {
	case <-ctx.Done():
		logging.Info("Context cancelled, exiting.")
	case <-sigChan:
		logging.Info("Signal received, exiting.")
	}
	logging.Info("Stopped!")
}

func Inject(ctx context.Context, port int, debug bool) {
	// 加载调用库
	log("Load dll:", libSdk)
	var err error
	gblDll, err = syscall.LoadDLL(libSdk)
	if err != nil {
		logging.ErrorWithErr(err, "Failed to load dll", map[string]interface{}{"hint": "请检查目录下是否放置sdk.dll & spy.dll & spy_debug.dll"})
		logging.Fatal("inject failed!", -1000)
	}

	logging.Info("### Inject SDK into WeChat ###")
	logging.Info(fmt.Sprintf("Set sdk port: %d, debug: %t", port, debug))

	startAt := time.Now()
	for {
		select {
		case <-ctx.Done():
			logging.Info("Injection process cancelled.")
			return
		default:
			if func() bool {
				defer func() {
					if r := recover(); r != nil { // 注入失败时反复重试
						logging.Warn(fmt.Sprintf("Get panic: %v, Wait for retry...", r))
						time.Sleep(3 * time.Second)
					}
				}()
				callFunc(funcInject, "Inject SDK...", debug, port)
				return true
			}() {
				goto InjectSuccess
			}
		}
	}
InjectSuccess:
	logging.Info(fmt.Sprintf("SDK inject success. Time used: %f", time.Now().Sub(startAt).Seconds()))
	waitingSignal(ctx)
	callFunc(funcDestroy, "SDK destroy", debug, port)
	_ = gblDll.Release()
}

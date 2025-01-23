// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/1/15 下午11:04:00
// @Desc 注入器
package wcf_rpc_sdk

import (
	"context"
	"fmt"
	"github.com/Clov614/logging"
	"io"
	"net/http"
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

func Inject(ctx context.Context, port int, debug bool, syncChan chan struct{}) {
	logging.Warn("自动注入中...", map[string]interface{}{"hint": "请检查是否安装对应微信3.9.11.25版本，如未安装请前往地址下载&安装", "wechatSetUpUrl": "https://github.com/lich0821/WeChatFerry/releases/download/v39.3.5/WeChatSetup-3.9.11.25.exe"})
	logging.Info("debug 模式状态", map[string]interface{}{"debug": debug})
	// 加载调用库
	log("Load dll:", libSdk)
	var err error
	gblDll, err = syscall.LoadDLL(libSdk)
	if err != nil {
		logging.ErrorWithErr(err, "Failed to load dll", map[string]interface{}{"hint": "请检查目录下是否放置sdk.dll & spy.dll & spy_debug.dll"})
		// 尝试下载并重试
		if !downloadAndRetry() {
			logging.Fatal("inject failed!", -1000)
		}
		return
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
			if tryInject(debug, port) {
				syncChan <- struct{}{} // 注入成功通知
				logging.Info(fmt.Sprintf("SDK inject success. Time used: %f", time.Now().Sub(startAt).Seconds()))
				waitingSignal(ctx)
				callFunc(funcDestroy, "SDK destroy", debug, port)
				_ = gblDll.Release()
				return
			}
		}
	}
}

// 尝试注入
func tryInject(debug bool, port int) (success bool) {
	defer func() {
		if r := recover(); r != nil { // 注入失败时反复重试
			logging.Error(fmt.Sprintf("Get panic: %v, Wait for retry...", r))
			time.Sleep(3 * time.Second)
			success = false
		}
	}()
	callFunc(funcInject, "Inject SDK...", debug, port)
	return true
}

// 下载所需 DLL 文件并重试注入
func downloadAndRetry() bool {
	dlls := []string{"sdk.dll", "spy.dll", "spy_debug.dll"}
	// 使用 raw.githubusercontent.com 的地址
	baseUrl := "https://raw.githubusercontent.com/Clov614/wcf-rpc-sdk/main/sources/sdk/"

	for _, dll := range dlls {
		url := baseUrl + dll
		logging.Info(fmt.Sprintf("Downloading %s from %s", dll, url))
		err := downloadFile(dll, url)
		if err != nil {
			logging.ErrorWithErr(err, fmt.Sprintf("Failed to download %s", dll), nil)
			return false
		}
		logging.Info(fmt.Sprintf("Successfully downloaded %s", dll))
	}

	// 重试注入
	logging.Fatal("已远程拉取.dll文件，请检查dll是否存在，重启程序", 0001)
	return false
}

// 下载文件的辅助函数
func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

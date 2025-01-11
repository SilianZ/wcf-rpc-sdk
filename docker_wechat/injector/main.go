package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// sdk.dll 的完整路径
const lib_sdk = "/root/sdk.dll"

// WxInitSDK 函数名
const func_inject = "WxInitSDK"

// WxDestroySDK 函数名
const func_destroy = "WxDestroySDK"

// 默认端口号
var gbl_port int = 8001

// 是否开启调试模式
var gbl_debug bool = true

// sdk.dll 的句柄
var gbl_dll *syscall.DLL

// 日志函数
func log(args ...interface{}) {
	fmt.Println("\033[1;7;32m[Inj]\033[0m", time.Now().Format("20060102_150405"), args)
}

/** 初始化. 加载动态库 */
func init() {
	log("Load dll:", lib_sdk)
	var err error
	// 加载 sdk.dll
	gbl_dll, err = syscall.LoadDLL(lib_sdk)
	if err != nil {
		panic(err)
	}
}

/** 调用库接口 */
func call_func(fun_name string, title string) {
	log(title)
	// log("Find function:", fun_name, "in dll:", gbl_dll)
	// 查找函数
	fun, err := gbl_dll.FindProc(fun_name)
	if err != nil {
		panic(err)
	}

	// log("Call function:", fun)
	// 设置调试模式参数
	dbgUintptr := uintptr(0)
	if gbl_debug {
		dbgUintptr = uintptr(1)
	}
	// 调用函数
	ret, _e, errno := syscall.Syscall(fun.Addr(), dbgUintptr, 0, uintptr(gbl_port), 0)
	if ret != 0 {
		panic("Function " + fmt.Sprint(fun) + " run failed! return:" +
			fmt.Sprint(ret) + ", err:" + fmt.Sprint(_e) + ", errno:" + fmt.Sprint(errno))
	}
}

/** 监听并等待SIGINT信号 */
func waiting_signal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	log("Is running, press Ctrl+C to quit.")
	<-sigChan
	log("Stopped!")
}

// 显示帮助信息
func show_help(ret int) {
	log("Usage:", os.Args[0], "[port [debug]]")
	os.Exit(ret)
}

func main() {
	log("### Inject SDK into WeChat ###")
	argc := len(os.Args)
	var err error
	// 解析命令行参数
	if argc > 1 {
		gbl_port, err = strconv.Atoi(os.Args[1])
		if err != nil {
			show_help(1)
		}
	}
	if argc > 2 {
		gbl_debug, err = strconv.ParseBool(os.Args[2])
		if err != nil {
			show_help(1)
		}
	}
	log("Set sdk port:", gbl_port, "debug:", gbl_debug)

	start_at := time.Now()
	for {
		if func() bool {
			defer func() {
				// 捕获 panic，防止注入失败导致程序退出
				if r := recover(); r != nil {
					log("Get panic:", r, " Wait for retry...")
					time.Sleep(3 * time.Second)
				}
			}()
			// 调用 WxInitSDK 函数注入 SDK
			call_func(func_inject, "Inject SDK...")
			return true
		}() {
			break
		}
	}
	log("SDK inject success. Time used:", time.Now().Sub(start_at).Seconds())
	// 等待 Ctrl+C 信号
	waiting_signal()
	// 调用 WxDestroySDK 函数销毁 SDK
	call_func(func_destroy, "SDK destroy")
	// 释放 sdk.dll
	gbl_dll.Release()
}

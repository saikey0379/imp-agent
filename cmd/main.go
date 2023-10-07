package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/wonderivan/craw"

	"github.com/saikey0379/imp-agent/pkg/agent"
	"github.com/saikey0379/imp-agent/pkg/build"
	"github.com/saikey0379/imp-agent/pkg/config"
	"github.com/saikey0379/imp-agent/pkg/config/iniconf"
	"github.com/saikey0379/imp-agent/pkg/logger"
	"github.com/saikey0379/imp-agent/pkg/utils"
)

const (
	DefaultCnf = "/../conf/imp-agent.conf"
)

type benchCraw struct {
}

func (this *benchCraw) Init() error {
	// 创建缓存时，在创建完成后会执行用户自定义的初始化函数，如果不需要初始化其他项，可以直接返回nil
	return nil
}

func (this *benchCraw) CustomGet(key string) (data interface{}, expired time.Duration, err error) {
	// 当调用craw获取远端数据时,内部会调用该方式实现,用户需要指定缓存数据过期时间，0为马上过期
	return key, -1, nil
}

func (this *benchCraw) CustomSet(key string, data interface{}) error {
	// 当调用craw设置远端数据时,内部会调用该方式实现，不需要设置，可以直接返回nil
	return nil
}

func (this *benchCraw) Destroy() {
	// 销毁缓存时，会执行用户自定义的销毁方法，如果不需要销毁其他项，该方法可以为空
}

func main() {
	app := cli.NewApp()
	app.Name = "imp-agent"
	app.Description = "imp agent"
	app.Version = build.Version()

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	configFile := dir + DefaultCnf
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Value:   configFile,
			Usage:   "config file",
		},
	}
	//创建监听退出chan
	c := make(chan os.Signal)
	//监听指定信号 ctrl+c kill
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				fmt.Println("退出", s)
				ExitFunc()
			case syscall.SIGUSR1:
				fmt.Println("usr1", s)
			case syscall.SIGUSR2:
				fmt.Println("usr2", s)
			default:
				fmt.Println("other", s)
			}
		}
	}()

	app.Action = func(c *cli.Context) (err error) {
		configFile = c.String("c")
		if !utils.FileExist(configFile) {
			return cli.NewExitError(fmt.Sprintf("The configuration file does not exist: %s", configFile), -1)
		}
		conf, err := iniconf.New(configFile).Load()
		if err = runAgent(conf, dir); err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
		return nil
	}
	app.Run(os.Args)
}

func runAgent(conf *config.Config, dir string) (err error) {
	log := logger.NewBeeLogger(conf)
	var agent = agent.NewAgent(log, conf, dir)
	agent.Craw = craw.NewCraw("agent", new(benchCraw))
	defer agent.Craw.Destroy()
	agent.Craw.SetCraw("md5_sysinfo", "", -1)
	//初始化agent
	if agent.Sn == "" {
		agent.Logger.Error("SN error:SN can not be empty!")
	}
	agent.Craw = agent.Craw
	agent.DaemonReportHardwareInfo()

	//OSinstall
	agent.DaemonPollingInstallInfo()

	//任务轮询
	agent.DaemonGetTaskBySn()
	//tcp监听
	lner, err := net.Listen("tcp", agent.ListenAddress)
	if err != nil {
		fmt.Println("Listener create error: ", err)
		return
	}
	for {
		conn, err := lner.Accept()
		if err != nil {
			fmt.Println("Accept error: ", err)
			return err
		}
		go agent.HandleConnection(conn)
	}
}

func ExitFunc() {
	fmt.Println("结束退出...")
	time.Sleep(time.Second * 1)
	os.Exit(0)
}

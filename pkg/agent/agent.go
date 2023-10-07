package agent

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/wonderivan/craw"

	"github.com/saikey0379/imp-agent/pkg/config"
	"github.com/saikey0379/imp-agent/pkg/logger"
)

const (
	DirTmpScripts    = "/tmp/imp-agent"
	PreInstallScript = DirTmpScripts + "/preInstall.sh"
	On               = "1"
)

var (
	defaultServerAddr       = "http://imp.example.com"
	defaultListenAddress    = "0.0.0.0:10079"
	defaultLoopInterval     = 300
	defaultTaskLoopInterval = 60
	defaultReportPolicy     = "always"

	defaultScriptGetSysInfo = "/../scripts/getSysInfo.sh"
	defaultScriptGetSn      = "/../scripts/getSn.sh"
	defaultScriptGetMac     = "/../scripts/getMac.sh"
	defaultScriptReboot     = "/../scripts/.reboot.sh"

	UrlReportSysInfo      = "/api/agent/reportSysInfo"
	UrlReportSysTimestamp = "/api/agent/reportSysTimestamp"
	URLReportMacInfo      = "/api/agent/reportMacInfo"
	UrlReportInstallInfo  = "/api/agent/reportInstallInfo"
	UrlReportTaskResult   = "/api/agent/reportTaskResult"

	UrlGetTaskFullListBySn   = "/api/agent/getTaskFullListBySn"
	UrlGetHardwareBySn       = "/api/agent/getHardwareBySn"
	UrlGetNetworkBySn        = "/api/agent/getNetworkBySn"
	UrlGetPrepareInstallInfo = "/api/agent/getPrepareInstallInfo"

	UrlIsInInstallList = "/api/agent/isInInstallList"
)

// Agent agent data struct
type Agent struct {
	Logger           logger.Logger
	Config           *config.Config
	Craw             *craw.Craw
	Sn               string
	ServerAddr       string
	ListenAddress    string
	LoopInterval     int
	TaskLoopInterval int
	PreScript        string
	DevelopeMode     string
	ReportPolicy     string
	hardwareConfs    []HardWareConf // base64 编码的硬件配置脚本
}

type nicInfo struct {
	Name string
	Mac  string
	Ip   string
}
type nicDevice struct {
	Id    string
	Model string
}
type cpuInfo struct {
	Id    string
	Model string
	Core  string
}
type diskInfo struct {
	Name string
	Size string
}
type memoryInfo struct {
	Name string
	Size string
	Type string
}
type gpuInfo struct {
	Id     string
	Model  string
	Memory string
}
type motherboardInfo struct {
	Name  string
	Model string
}

// New create agent
func NewAgent(log logger.Logger, conf *config.Config, dir string) *Agent {
	// get config
	var agent = &Agent{
		Config: conf,
		Logger: log,
	}

	// get server addr
	if conf.Agent.Server == "" {
		agent.ServerAddr = defaultServerAddr
	} else {
		agent.ServerAddr = conf.Agent.Server
	}
	agent.ServerAddr = strings.Trim(agent.ServerAddr, "\n")
	defaultServerAddr = agent.ServerAddr
	agent.Logger.Debugf("SERVER_ADDR: %s", agent.ServerAddr)

	//agent listen
	if conf.Agent.Listen == "" {
		agent.ListenAddress = defaultListenAddress
	} else {
		agent.ListenAddress = conf.Agent.Listen
	}
	agent.Logger.Debugf("ListenAddress: %s", agent.ListenAddress)

	// loop interval
	if conf.Agent.Interval == 0 {
		agent.LoopInterval = defaultLoopInterval
	} else {
		agent.LoopInterval = conf.Agent.Interval
	}
	agent.Logger.Debugf("LOOP_INTERVAL: %s", strconv.Itoa(agent.LoopInterval))

	// task loop interval
	agent.TaskLoopInterval = defaultTaskLoopInterval

	// developMode
	agent.DevelopeMode = strconv.Itoa(conf.Agent.Developer)
	agent.DevelopeMode = strings.Trim(agent.DevelopeMode, "\n")
	agent.Logger.Debugf("DEVELOPER: %s", agent.DevelopeMode)

	// preScript
	agent.PreScript = conf.Agent.PreScript
	if agent.PreScript != "" {
		agent.Logger.Debugf("PRE_SCRIPT: %s", agent.PreScript)
	}

	// ReportPolicy
	if conf.Agent.ReportPolicy == "" {
		agent.ReportPolicy = defaultReportPolicy
	} else {
		agent.ReportPolicy = conf.Agent.ReportPolicy
	}
	agent.ReportPolicy = strings.Trim(agent.ReportPolicy, "\n")
	agent.Logger.Debugf("ReportPolicy: %s", agent.ReportPolicy)

	return agent
}

func wget(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// execScript 执行脚本
func execScript(script string) ([]byte, error) {

	// 生成临时文件
	file, err := ioutil.TempFile("", "tmp-script")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	defer file.Close()

	if _, err = file.WriteString(script); err != nil {
		return nil, err
	}
	file.Close()

	var cmd = exec.Command("/bin/bash", file.Name())
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	err = cmd.Wait()
	return output.Bytes(), err
}

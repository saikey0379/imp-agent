package agent

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/saikey0379/imp-agent/pkg/utils"
)

// HardWareConf 硬件配置结构
type HardWareConf struct {
	Name    string
	Scripts []struct {
		Name   string
		Script string
	}
}

func (agent *Agent) DaemonPollingInstallInfo() {
	go agent.PollingInstallInfo()
}

func (agent *Agent) PollingInstallInfo() {
	for {
		var err error
		// 状态轮询（是否在装机队列中）
		agent.IsInInstallQueue()
		agent.Logger.Debug("into install queue")
		agent.ReportProgress(0.05, "进入bootos", "正常进入bootos")
		// 配置查询（15%）
		var isSkip = false
		isSkip, err = agent.IsHaveHardWareConf()
		if !isSkip {
			if err != nil {
				agent.ReportProgress(-1, "配置查询失败", "该硬件型号不存在，请打开开发者模式再尝试，错误信息："+err.Error())
				continue
			} else {
				agent.ReportProgress(0.08, "配置查询", "存在对应的硬件配置")
			}
			// 获取硬件配置模板（10%）
			if err = agent.GetHardWareConf(); err != nil {
				agent.ReportProgress(-1, "获取硬件配置模板失败", "没有对应的硬件配置模板，错误信息："+err.Error())
				continue
			} else {
				agent.ReportProgress(0.1, "获取硬件配置", "存在对应的硬件配置模板")
			}
			// 硬件初始化（10%~15%）
			if err = agent.ImplementHardConf(); err != nil {
				agent.ReportProgress(-1, "初始化硬件失败", "无法初始化硬件，错误信息："+err.Error())
				continue
			}
		}
		// 生成 PXE文件（15%）
		if err = agent.ReportMacInfo(); err != nil {
			agent.ReportProgress(-1, "生成PXE文件失败", "无法生成PXE文件，错误信息："+err.Error())
			continue
		} else {
			agent.ReportProgress(0.15, "生成PXE文件", "正常生成PXE文件")
		}
		//run pre install script
		agent.RunPreInstallScript()
		// 重启系统（20%）
		agent.ReportProgress(0.2, "系统开始重启", "系统重启中... ...")
		if err = agent.Reboot(); err != nil {
			agent.ReportProgress(-1, "系统重启失败", "重启系统出错，错误信息："+err.Error())
			continue
		} else {
			break // 退出 agent
		}
	}
}

// IsInInstallQueue 检查是否在装机队列中 （定时执行）
func (agent *Agent) IsInInstallQueue() {
	// 轮询是否在装机队列中
	var t = time.NewTicker(time.Duration(agent.LoopInterval) * time.Second)
	var url = agent.ServerAddr + UrlIsInInstallList
	agent.Logger.Debugf("IsInPreInstallQueue url:%s", url)
	var jsonReq struct {
		Sn string
	}
	jsonReq.Sn = agent.Sn
LOOP:
	for {
		agent.Logger.Debugf("IsInPreInstallQueue request body: %v", jsonReq)
		var ret, err = utils.CallRestAPI(url, jsonReq)
		agent.Logger.Debugf("IsInPreInstallQueue api result: %s", strings.Replace(strings.Replace(string(ret), "\n", "", -1), " ", "", -1))
		if err != nil {
			agent.Logger.Error(err)
			<-t.C
			continue // 继续等待下次轮询
		}
		var jsonResp struct {
			Status  string
			Message string
			Content struct {
				Result string
			}
		}
		if err := json.Unmarshal(ret, &jsonResp); err != nil {
			agent.Logger.Error(err)
			<-t.C
			continue // 继续等待下次轮询
		}
		if jsonResp.Content.Result == "true" {
			t.Stop()
			break LOOP
		}
		<-t.C
	}
}

// IsHaveHardWareConf 检查服务端是否此机器的硬件配置
func (agent *Agent) IsHaveHardWareConf() (bool, error) {
	var url = agent.ServerAddr + UrlGetPrepareInstallInfo
	var skipHWConf = false
	agent.Logger.Debugf("IsHaveHardWareConf url:%s\n", url)
	var jsonReq struct {
		Sn        string
		Company   string
		Product   string
		ModelName string
	}
	jsonReq.Sn = agent.Sn

	var jsonResp struct {
		Status  string
		Message string
		Content struct {
			IsVerify             string
			IsSkipHardwareConfig string
		}
	}
	agent.Logger.Debugf("IsHaveHardWareConf request body: %v", jsonReq)
	var ret, err = utils.CallRestAPI(url, jsonReq)
	agent.Logger.Debugf("IsHaveHardWareConf api result:%s\n", string(ret))
	if err != nil {
		return skipHWConf, err
	}
	if err := json.Unmarshal(ret, &jsonResp); err != nil {
		return skipHWConf, err
	}
	if jsonResp.Status != "success" {
		return skipHWConf, fmt.Errorf("Status: %s, Message: %s", jsonResp.Status, jsonResp.Message)
	}
	// is skip hardware configuration
	if jsonResp.Content.IsSkipHardwareConfig == "true" {
		return true, nil
	}
	if jsonResp.Content.IsVerify == "false" && agent.DevelopeMode != On {
		return skipHWConf, errors.New("Verify is false AND developMode is not On")
	}
	return false, nil
}

// GetHardConf 获取硬件配置
func (agent *Agent) GetHardWareConf() error {
	var url = agent.ServerAddr + UrlGetHardwareBySn
	agent.Logger.Debugf("GetHardWareConf url:%s\n", url)
	var jsonReq struct {
		Sn string
	}
	jsonReq.Sn = agent.Sn
	var jsonResp struct {
		Status  string
		Message string
		Content struct {
			Company   string
			ModelName string
			Product   string
			Hardware  []HardWareConf
		}
	}
	agent.Logger.Debugf("GetHardWareConf request body: %v", jsonReq)
	var ret, err = utils.CallRestAPI(url, jsonReq)
	agent.Logger.Debugf("GetHardWareConf api result:%s\n", string(ret))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(ret, &jsonResp); err != nil {
		return err
	}
	if jsonResp.Status != "success" {
		return fmt.Errorf("Status: %s, Message: %s", jsonResp.Status, jsonResp.Message)
	}
	agent.hardwareConfs = jsonResp.Content.Hardware
	return nil
}

// ImplementHardConf 实施硬件配置
func (agent *Agent) ImplementHardConf() error {
	// 开始硬件配置
	agent.ReportProgress(0.3, "开始硬件配置", "")
	var progressDelta int
	if len(agent.hardwareConfs) != 0 {
		progressDelta = 10 / len(agent.hardwareConfs)
	}
	var curProgress = 0.3
	for _, hardwareConf := range agent.hardwareConfs {
		curProgress = curProgress + float64(progressDelta)/100.0
		for _, scriptB64 := range hardwareConf.Scripts {
			script, err := base64.StdEncoding.DecodeString(scriptB64.Script)
			agent.Logger.Debugf("Script: %s\n", script)
			if err != nil {
				return err
			}
			if output, err := execScript(string(script)); err != nil {
				return fmt.Errorf("execscript hardware script error: \n#%s\n%v\n%s", string(script), err, string(output))
			}
			agent.ReportProgress(curProgress, hardwareConf.Name+" - "+scriptB64.Name, "")
		}
		agent.ReportProgress(curProgress, hardwareConf.Name+" 配置完成", "")
	}
	agent.ReportProgress(0.4, "硬件配置结束", "硬件配置正常结束")
	return nil
}

// ReportProgress 上报执行结果
func (agent *Agent) ReportProgress(installProgress float64, title, installLog string) bool {
	var url = agent.ServerAddr + UrlReportInstallInfo
	agent.Logger.Debugf("ReportProgress url:%s\n", url)
	var jsonReq struct {
		Sn              string
		InstallProgress float64
		InstallLog      string
		Title           string
	}
	jsonReq.Sn = agent.Sn
	jsonReq.InstallProgress = installProgress
	jsonReq.Title = title
	jsonReq.InstallLog = base64.StdEncoding.EncodeToString([]byte(installLog)) // base64编码
	agent.Logger.Debugf("SN: %s\n", jsonReq.Sn)
	agent.Logger.Debugf("InstallProgress: %f\n", jsonReq.InstallProgress)
	agent.Logger.Debugf("InstallLog: %s\n", jsonReq.InstallLog)
	agent.Logger.Debugf("Title: %s\n", jsonReq.Title)
	var jsonResp struct {
		Status  string
		Message string
		Content struct {
			Result string
		}
	}
	agent.Logger.Debugf("ReportProgress request body: %v", jsonReq)
	var ret, err = utils.CallRestAPI(url, jsonReq)
	agent.Logger.Debugf("ReportProgress api result:%s\n", string(ret))
	if err != nil {
		agent.Logger.Error(err.Error())
		return false
	}
	if err := json.Unmarshal(ret, &jsonResp); err != nil {
		agent.Logger.Error(err.Error())
		return false
	}
	if jsonResp.Status != "success" {
		return false
	}
	return true
}

// ReportMacInfo 上报 mac 地址
func (agent *Agent) ReportMacInfo() error {
	var url = agent.ServerAddr + URLReportMacInfo
	agent.Logger.Debugf("ReportMacInfo url:%s\n", url)
	var jsonReq struct {
		Sn  string
		Mac []string
	}
	jsonReq.Sn = agent.Sn

	data, err := execScript(defaultScriptGetMac)
	if err != nil {
		agent.Logger.Debugf("Get Mac Addr Error: %s", err)
		return err
	}
	for _, i := range strings.Split(string(data), "\n") {
		if i != "" {
			jsonReq.Mac = append(jsonReq.Mac, i)
		}
	}
	agent.Logger.Debugf("Mac ADDR: %s", jsonReq.Mac)

	var jsonResp struct {
		Status  string
		Message string
		Content struct {
			Result string
		}
	}
	var ret []byte
	agent.Logger.Debugf("ReportMacInfo request body: %v", jsonReq)
	ret, err = utils.CallRestAPI(url, jsonReq)
	agent.Logger.Debugf("ReportMacInfo api result:%s\n", string(ret))
	if err != nil {
		return err
	}

	if err := json.Unmarshal(ret, &jsonResp); err != nil {
		return err
	}

	if jsonResp.Status != "success" {
		return fmt.Errorf("Status: %s, Message: %s", jsonResp.Status, jsonResp.Message)
	}
	return nil
}

// run pre install script
func (agent *Agent) RunPreInstallScript() {
	url := agent.PreScript
	if url == "" {
		agent.Logger.Debugf("PreInstallScript: not config")
		return
	}
	agent.Logger.Debugf("PreInstallscript: %s", agent.PreScript)
	url = strings.Trim(url, "\n")
	url = strings.TrimSpace(url)

	agent.Logger.Infof("script:%s", url)
	if url == "" {
		return
	}

	agent.Logger.Debugf("START to wget %s", url)
	script, err := wget(url)
	if err != nil {
		agent.Logger.Error(err.Error())
		return
	}
	agent.Logger.Debugf("script:%s", script)

	agent.Logger.Debugf("write to file %s:%s", PreInstallScript, script)
	var bytes = []byte(script)
	errWrite := ioutil.WriteFile(PreInstallScript, bytes, 0666)
	if errWrite != nil {
		agent.Logger.Error(errWrite.Error())
		return
	}

	//chmod 755 PreInstallScript
	cmd := `chmod 755 ` + PreInstallScript
	agent.Logger.Debugf("exec:%s", cmd)
	data, errRun := execScript(cmd)
	if errRun != nil {
		agent.Logger.Error(errRun.Error())
		return
	}
	agent.Logger.Debugf("result:%s", string(data))

	//run PreInstallScript
	cmd = PreInstallScript
	agent.Logger.Debugf("exec:%s", cmd)
	data, errRun = execScript(cmd)
	if errRun != nil {
		agent.Logger.Error(errRun.Error())
		return
	}
	agent.Logger.Debugf("result:%s", string(data))
	return
}

// Reboot 重启系统
func (agent *Agent) Reboot() error {
	if output, err := execScript(defaultScriptReboot); err != nil {
		return fmt.Errorf("reboot error: \n#%s\n%v\n%s\n\n", defaultScriptReboot, err, string(output))
	}
	return nil
}

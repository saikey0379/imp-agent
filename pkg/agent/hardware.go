package agent

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/saikey0379/imp-agent/pkg/utils"
)

func (agent *Agent) DaemonReportHardwareInfo() {
	go agent.ReportHardwareInfo()
}

// ReportHardwareInfo 周期上报deviceInfo （定时执行）
func (agent *Agent) ReportHardwareInfo() {
	// 轮询是否在装机队列中
	var t = time.NewTicker(time.Duration(agent.LoopInterval) * time.Second)

	for {
		if err := agent.ReportProductInfo(); err != nil {
			agent.Logger.Error(fmt.Sprintf("ERROR: ReportProductInfo[%s]", err.Error()))
		}
		<-t.C
	}
}

func (agent *Agent) ReportProductInfo() error {
	sysInfo, err := agent.GetProductInfo()
	if err != nil {
		agent.Logger.Errorf(fmt.Sprintf("ERROR: GetProductInfo", err.Error()))
		return err
	}

	var changed = true
	if agent.ReportPolicy == "changed" {
		//diff sysinfo md5 in craw and scripts output
		changed, err = agent.IsProductInfoChanged(sysInfo)
		agent.Logger.Errorf(fmt.Sprintf("ERROR: IsProductInfoChanged", err.Error()))
		return err
	}

	if changed {
		err = agent.UpdateProductInfo(sysInfo)
		if err != nil {
			agent.Logger.Errorf(fmt.Sprintf("ERROR: UpdateProductInfo", err.Error()))
			return err
		}
	} else {
		err = agent.UpdateTimestamp()
		if err != nil {
			agent.Logger.Errorf(fmt.Sprintf("ERROR: UpdateTimestamp", err.Error()))
			return err
		}
	}
	return err
}

func (agent *Agent) UpdateProductInfo(sysInfo []byte) (err error) {
	var jsonReq struct {
		Sn          string
		Company     string
		ModelName   string
		Motherboard motherboardInfo
		Raid        string
		NicDevice   []nicDevice
		Oob         string
		Cpu         []cpuInfo
		CpuSum      int
		Memory      []memoryInfo
		MemorySum   int
		DiskSum     int
		Nic         []nicInfo
		Disk        []diskInfo
		Gpu         []gpuInfo
		IsVm        string
		VersionAgt  string
	}

	if err = json.Unmarshal(sysInfo, &jsonReq); err != nil {
		return err
	}
	agent.Logger.Debugf("ReportProductInfo request body: %v", jsonReq)

	//start to ReportProductInfo
	var url = agent.ServerAddr + UrlReportSysInfo
	agent.Logger.Debugf("ReportProductInfo url:%s", url)
	// set company to agent
	agent.Sn = strings.Trim(jsonReq.Sn, "\n")
	agent.Sn = strings.TrimSpace(agent.Sn)
	agent.Logger.Debugf("ProductInfoSN: %s", agent.Sn)

	var jsonResp struct {
		Status  string
		Message string
	}

	ret, err := utils.CallRestAPI(url, jsonReq)
	if err != nil {
		return err
	}
	agent.Logger.Debugf("ReportProductInfo api result:%s", strings.Replace(string(ret), "\n", "", -1))

	if err = json.Unmarshal(ret, &jsonResp); err != nil {
		return err
	}
	if jsonResp.Status != "success" {
		return fmt.Errorf("Status: %s, Message: %s", jsonResp.Status, jsonResp.Message)
	}
	return nil
}

func (agent *Agent) UpdateTimestamp() (err error) {
	var jsonReq struct {
		Sn string
	}
	var url = agent.ServerAddr + UrlReportSysTimestamp
	jsonReq.Sn = agent.Sn
	agent.Logger.Debugf("ReportProductInfo request body: %v", jsonReq)

	var jsonResp struct {
		Status  string
		Message string
	}
	ret, err := utils.CallRestAPI(url, jsonReq)
	if err != nil {
		return err
	}
	agent.Logger.Debugf("ReportProductInfo api result:%s", strings.Replace(string(ret), "\n", "", -1))

	if err = json.Unmarshal(ret, &jsonResp); err != nil {
		return err
	}
	if jsonResp.Status != "success" {
		return fmt.Errorf("Status: %s, Message: %s", jsonResp.Status, jsonResp.Message)
	}
	return nil
}

func (agent *Agent) IsProductInfoChanged(sysInfo []byte) (change bool, err error) {
	//diff sysinfo md5 in craw and scripts output
	md5_sysinfo_craw, err := agent.Craw.GetData("md5_sysinfo")
	if err != nil {
		agent.Logger.Errorf("ERROR: ReportProductInfo: SysInfo not changed")
		return true, err
	}

	md5sum_req := md5.Sum(sysInfo)
	md5_sysinfo_curr := hex.EncodeToString(md5sum_req[:])
	if md5_sysinfo_curr == md5_sysinfo_craw {
		agent.Logger.Debugf("ReportProductInfo: SysInfo not changed")
		return false, err
	}
	err = agent.Craw.SetCraw("md5_sysinfo", md5_sysinfo_curr, -1)
	return true, err
}

func (agent *Agent) GetProductInfo() (sysInfo []byte, err error) {
	// get infoFull from script
	fileMod, err := utils.GetFileMod(defaultScriptGetSysInfo)
	if err != nil {
		return sysInfo, err
	}

	if fileMod != os.FileMode(493) {
		err = os.Chmod(defaultScriptGetSysInfo, os.FileMode(493))
		if err != nil {
			return sysInfo, err
		}
	}
	sysInfo, err = execScript(defaultScriptGetSysInfo)
	if err != nil {
		return sysInfo, fmt.Errorf("ReportProductInfo error: \n#%s\n%v\n%s", defaultScriptGetSysInfo, err, string(sysInfo))
	}
	return sysInfo, err
}

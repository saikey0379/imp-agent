package agent

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron"
	"github.com/wonderivan/craw"

	"github.com/saikey0379/imp-agent/pkg/logger"
	"github.com/saikey0379/imp-agent/pkg/utils"
)

type TaskInfo struct {
	ID         int
	BatchId    int
	TaskType   string
	TaskPolicy string
	ReportURL  string
	Logger     logger.Logger
	File       FileInfo
}

func (agent *Agent) DaemonGetTaskBySn() {
	go agent.GetTaskBySn()
}

// 触发、同步任务监听
func (agent *Agent) HandleConnection(conn net.Conn) {
	clientAddr := conn.RemoteAddr().String()
	agent.Logger.Info("Connection success: " + clientAddr)

	defer conn.Close()
	var message bytes.Buffer

	buffer := make([]byte, 10240)
	recvLen, err := conn.Read(buffer)
	message.Write(buffer[:recvLen])

	if err != nil {
		agent.Logger.Error("HandleConnRead: ", err, clientAddr)
		return
	}
	var taskInfo TaskInfo

	agent.Logger.Debug("HandleMessageInfo: " + message.String())
	agent.Logger.Debug("HandleClientAddr: " + clientAddr)
	if err := json.Unmarshal(message.Bytes(), &taskInfo); err != nil {
		agent.Logger.Error(err)
		return
	}

	taskInfo.ReportURL = agent.ServerAddr + UrlReportTaskResult
	agent.Logger.Debug("HandleTaskReport url: " + taskInfo.ReportURL)

	err = taskInfo.RunTask()
	if err != nil {
		agent.Logger.Error("RunTask: ", err)
		_, err = conn.Write([]byte(err.Error()))
		if err != nil {
			agent.Logger.Error("HandleConnection: Write", err)
			return
		}
	} else {
		_, err = conn.Write([]byte("success"))
		if err != nil {
			agent.Logger.Error("HandleConnection: Write", err)
			return
		}
	}
}

// 任务查询
func (agent *Agent) GetTaskBySn() {
	// 轮询获取任务列表
	var t = time.NewTicker(time.Duration(agent.TaskLoopInterval) * time.Second)
	var url = agent.ServerAddr + UrlGetTaskFullListBySn
	agent.Logger.Debugf("GetTaskBySn url:%s", url)
	var jsonReq struct {
		Sn string
	}
	jsonReq.Sn = agent.Sn
	for {
		<-t.C
		agent.Logger.Debugf("GetTaskBySn request body: %v", jsonReq)
		reqByte, err := json.Marshal(jsonReq)
		if err != nil {
			agent.Logger.Errorf("GetTaskBySn json.Marshal(orderInform):[%s]", err.Error())
		}

		ret, err := utils.PostRestApi(url, reqByte)
		agent.Logger.Debugf("GetTaskBySn api result: %s", strings.Replace(strings.Replace(string(ret), "\n", "", -1), " ", "", -1))
		agent.Craw.SetCraw("task_status", "", -1)
		if err != nil {
			agent.Logger.Error(err)
			agent.Craw.SetCraw("task_status", "stop", -1)
			continue // 继续等待下次轮询
		}
		var jsonResp struct {
			Status  string
			Message string
			Content []TaskInfo
		}
		if err := json.Unmarshal(ret, &jsonResp); err != nil {
			agent.Logger.Error(err)
			agent.Craw.SetCraw("task_status", "stop", -1)
			continue // 继续等待下次轮询
		}
		agent.Logger.Debugf("GetTaskBySn content: %v", jsonResp.Content)
		if len(jsonResp.Content) == 0 {
			agent.Craw.SetCraw("task_ids", "", -1)
			agent.Craw.SetCraw("task_status", "stop", -1)
			continue // 继续等待下次轮询
		}
		//设置计划任务
		var task_ids string
		for i, v := range jsonResp.Content {
			id := strconv.Itoa(v.ID)
			if i == 0 {
				task_ids = id
			} else {
				task_ids = task_ids + "\n" + id
			}

		}
		agent.Craw.SetCraw("task_ids", task_ids, -1)
		agent.Logger.Debugf("GetTaskBySn task_ids: %s", task_ids)

		for _, taskInfo := range jsonResp.Content {
			taskInfo.ReportURL = agent.ServerAddr + UrlReportTaskResult
			taskInfo.Logger = agent.Logger

			id := strconv.Itoa(taskInfo.ID)
			md5_last, _ := agent.Craw.GetData(id)
			md5sum_task := md5.Sum([]byte(fmt.Sprintf("%+v", taskInfo)))
			md5_curr := hex.EncodeToString(md5sum_task[:])
			var cron_start bool
			if md5_curr != md5_last {
				agent.Craw.SetCraw(id, md5_curr, -1)
				cron_start = true
			}
			switch taskInfo.TaskType {
			case "cron":
				if cron_start {
					go RunCronTask(taskInfo, agent.Craw)
				}
				continue
			default:
				go RunFixedTask(taskInfo)
				continue
			}
		}
	}
}

// 即时任务与定时任务
func RunFixedTask(task TaskInfo) {
	time_now := time.Now().Format("2006-01-02 15:04")
	if task.TaskPolicy == time_now {
		t := &task
		t.Logger.Debugf("TaskFixedStart task: %s", task)
		t.RunTaskDeamon()
	}
	return
}

// 计划任务
func RunCronTask(task TaskInfo, craw *craw.Craw) {
	id := strconv.Itoa(task.ID)
	md5sum_task := md5.Sum([]byte(fmt.Sprintf("%+v", task)))
	md5_start := hex.EncodeToString(md5sum_task[:])

	cronPolicy := "00 " + strings.Replace(task.TaskPolicy, ",", " ", -1)
	c := cron.New()
	t := &task
	c.AddFunc(cronPolicy, t.RunTaskDeamon)

	t.Logger.Debugf("TaskCronStart task: %s", task)
	c.Start()

	var tc = time.NewTicker(time.Duration(1) * time.Second)
	//LOOP:
	for {
		select {
		case <-tc.C:
			md5_curr, _ := craw.GetData(id)
			ids, _ := craw.GetData("task_ids")
			status, _ := craw.GetData("task_status")
			if !strings.Contains(fmt.Sprintf("%v", ids), id) || status == "stop" {
				craw.ClearPrefixKeys(id)
				t.Logger.Info("Clear task: " + id)
				c.Stop()
				return
			}
			if md5_curr != md5_start {
				c.Stop()
				return
			}
		}
	}
}

// 触发任务/同步任务执行
func (task *TaskInfo) RunTask() error {
	var jsonReq struct {
		TaskId    int
		BatchId   int
		Hostname  string
		FileSync  string
		Result    string
		Status    string
		StartTime time.Time
		EndTime   time.Time
	}
	jsonReq.TaskId = task.ID
	jsonReq.BatchId = task.BatchId
	jsonReq.Hostname, _ = os.Hostname()

	jsonReq.StartTime = time.Now()

	var results []string
	//TaskFileRsync
	filesync, err := task.File.FileRsync()
	if err != nil {
		filesync = append(filesync, err.Error())
		goto REPORT
	}
	//RunTask
	switch task.File.FileType {
	case "script", "execution":
		var result string
		result, err = task.File.FileExec()

		results = append(results, result)
		if err != nil {
			results = append(results, err.Error())
		}
	}

REPORT:
	jsonReq.FileSync = strings.Join(filesync, "\n")
	jsonReq.Result = strings.Join(results, "\n")
	if err != nil {
		jsonReq.Status = "failure"
		fmt.Println("ERROR: RunTask Result:[" + jsonReq.Result + "]")
	} else {
		jsonReq.Status = "success"
		fmt.Println("SUCCESS: RunTask Result:[" + jsonReq.Result + "]")
	}
	jsonReq.EndTime = time.Now()
	reqByte, err := json.Marshal(jsonReq)
	if err != nil {
		fmt.Println("ERROR: RunTask jsonReq.Marshal[" + err.Error() + "]")
		return err
	}
	resp, err := utils.PostRestApi(task.ReportURL, reqByte)
	if err != nil {
		fmt.Println("ERROR: RunTask PostRestApi[" + err.Error() + "]")
	} else {
		fmt.Println("SUCCESS: RunTask PostRestApi[" + string(resp) + "]")
	}
	return err
}

// 异步任务执行
func (task *TaskInfo) RunTaskDeamon() {
	var jsonReq struct {
		TaskId    int
		BatchId   int
		Hostname  string
		FileSync  string
		Result    string
		Status    string
		StartTime time.Time
		EndTime   time.Time
	}
	jsonReq.TaskId = task.ID
	jsonReq.Hostname, _ = os.Hostname()
	batchid := time.Now().Format("200601021504")
	jsonReq.BatchId, _ = strconv.Atoi(batchid)

	jsonReq.StartTime = time.Now()

	var results []string
	//TaskFileRsync
	filesync, err := task.File.FileRsync()
	if err != nil {
		task.Logger.Error("TaskFileRsync: " + err.Error())
		filesync = append(filesync, err.Error())
		goto REPORT
	}
	task.Logger.Debugf("TaskFileRsync: " + strings.Join(filesync, ","))

	//RunTask
	switch task.File.FileType {
	case "script", "execution":
		var result string
		result, err = task.File.FileExec()
		results = append(results, result)
		if err != nil {
			task.Logger.Error("TaskFileExec: " + err.Error())
			results = append(results, err.Error())
		}
		task.Logger.Debugf("TaskFileExec: " + strings.Join(results, ","))
	}

REPORT:
	jsonReq.FileSync = strings.Join(filesync, "\n")
	jsonReq.Result = strings.Join(results, "\n")
	if err != nil {
		jsonReq.Status = "failure"
		fmt.Println("ERROR: RunTaskDeamon Result:[" + jsonReq.Result + "]")
	} else {
		jsonReq.Status = "success"
		fmt.Println("SUCCESS: RunTaskDeamon Result:[" + jsonReq.Result + "]")
	}
	jsonReq.EndTime = time.Now()
	reqByte, err := json.Marshal(jsonReq)
	if err != nil {
		task.Logger.Error("TaskResult Marshal: " + err.Error())
		fmt.Println("ERROR: RunTaskDeamon jsonReq.Marshal[" + err.Error() + "]")
	}
	resp, err := utils.PostRestApi(task.ReportURL, reqByte)
	if err != nil {
		task.Logger.Error("TaskResult Post: " + err.Error())
		fmt.Println("ERROR: RunTaskDeamon PostRestApi[" + err.Error() + "]")
	} else {
		task.Logger.Debugf("TaskResult Post: " + string(resp))
		fmt.Println("SUCCESS: RunTaskDeamon PostRestApi[" + string(resp) + "]")
	}
	return
}

package agent

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/saikey0379/imp-agent/pkg/utils"
)

type FileInfo struct {
	FileId      uint
	FileName    string
	FileType    string
	FileLink    string
	FileMod     string
	Interpreter string
	Parameter   string
	DestPath    string
	Md5         string
}

// 文件摘要及权限校验与文件同步
func (file *FileInfo) FileRsync() ([]string, error) {
	var results []string
	var err error
	//文件下载channel及链接地址
	var downloadLink = defaultServerAddr + file.FileLink
	//文件权限
	fm, err := strconv.ParseUint(file.FileMod, 8, 32)
	if err != nil {
		return results, err
	}
	var taskfileMod = os.FileMode(fm)

	//确定文件信息
	var fileInfo struct {
		FileName string
		FileDest string
		FileTemp string
		FileMeta string
	}

	if file.DestPath != "" {
		fileInfo.FileDest = file.DestPath
	} else {
		fileInfo.FileDest = fmt.Sprintf("%s/%s", DirTmpScripts, file.FileName)
	}
	fileInfo.FileName = fileInfo.FileDest[strings.LastIndex(fileInfo.FileDest, "/")+1:]

	dir := fileInfo.FileDest[0 : strings.LastIndex(fileInfo.FileDest, "/")+1]
	fileInfo.FileTemp = dir + "." + fileInfo.FileName
	fileInfo.FileMeta = fileInfo.FileTemp + "_meta"

	//目录检查并创建
	_, err = os.Stat(dir)
	if err != nil {
		// 创建文件夹
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return results, err
		}
	}

	var boolRFile bool
	var boolRMode bool

	var fileMd5Now string
	var fileModNow os.FileMode
	if utils.FileExist(fileInfo.FileDest) {
		//文件变更，备份
		fileMd5Now, err = utils.GetFileMd5(fileInfo.FileDest)
		if err != nil {
			return results, err
		}
		if fileMd5Now != file.Md5 {
			boolRFile = true
			err = utils.FileBackup(fileInfo.FileDest, fmt.Sprintf("%s/bak", DirTmpScripts))
			if err != nil {
				return results, err
			}
		} else {
			//文件权限
			fileModNow, err = utils.GetFileMod(fileInfo.FileDest)
			if err != nil {
				return results, err
			}
			if fileModNow != taskfileMod {
				boolRMode = true
				err = os.Chmod(fileInfo.FileDest, taskfileMod)
				if err != nil {
					return results, err
				}
			} else {
				results = append(results, "NOTICE: File Not Changed")
				return results, err
			}
		}
	} else {
		boolRFile = true
	}

	//确认文件是否需要更新
	if boolRFile {
		//下载临时文件
		err = utils.FileDownload(downloadLink, fileInfo.FileTemp, taskfileMod)
		if err != nil {
			return results, err
		}
		//获取MD5
		md5New, err := utils.GetFileMd5(fileInfo.FileTemp)
		if err != nil {
			return results, err
		}
		if md5New != file.Md5 {
			return results, fmt.Errorf("ERROR:FileDest MD5 Not Matched[Expected:" + file.Md5 + "/Current:" + md5New)
		}
		//更新File
		err = os.Rename(fileInfo.FileTemp, fileInfo.FileDest)
		if err != nil {
			return results, err
		}
	}

	results = append(results, "File: "+fileInfo.FileDest)
	//filemeta_now
	var filemeta_now string
	if utils.FileExist(fileInfo.FileMeta) {
		bytes, err := ioutil.ReadFile(fileInfo.FileMeta)
		if err != nil {
			return results, err
		}
		filemeta_now = string(bytes)
	}

	//filemeta_new
	if boolRFile || boolRMode {
		filemeta_new := "ID[" + strconv.FormatUint(uint64(file.FileId), 10) + "] MD5[" + file.Md5 + "] Mod[" + file.FileMod + "] Name[" + file.FileName + "]"
		err = utils.FileCreate(filemeta_new, fileInfo.FileMeta)
		if err != nil {
			return results, err
		}
		results = append(results, "NEW: "+filemeta_new)
	}

	if filemeta_now != "" {
		results = append(results, "OLD: "+filemeta_now)
	} else {
		var result string
		if fileMd5Now != "" {
			result = "OLD: MD5[" + fileMd5Now + "] "
		}
		if boolRMode {
			if result == "" {
				result = "OLD: Mod[" + fileModNow.String() + "]"
			} else {
				result = result + "Mod[" + fileModNow.String() + "]"
			}
		}
		results = append(results, result)
	}

	utils.DropPageCache(file.DestPath)

	return results, err
}

// 任务执行
func (file *FileInfo) FileExec() (string, error) {
	var err error
	var fileDest string
	if file.DestPath != "" {
		fileDest = file.DestPath
	} else {
		fileDest = fmt.Sprintf("%s/%s", DirTmpScripts, file.FileName)
	}

	var args []string
	var cmd *exec.Cmd
	switch file.FileType {
	case "script":
		args = strings.Split(fmt.Sprintf("%s %s", fileDest, file.Parameter), " ")
		cmd = exec.Command(file.Interpreter, args[0:]...)
	case "execution":
		args = strings.Split(file.Parameter, " ")
		cmd = exec.Command(fileDest, args[0:]...)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	utils.DropPageCache(file.DestPath)

	return string(out.Bytes()), err
}

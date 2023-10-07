package utils

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func FileBackup(fileSrc, backupDir string) error {
	var results []string
	var err error
	//备份目录检查并创建
	_, err = os.Stat(backupDir)
	if err != nil {
		results = append(results, "ERROR:FileBakDir Stats "+err.Error())
	}
	if os.IsNotExist(err) {
		// 创建文件夹
		err = os.MkdirAll(backupDir, os.ModePerm)
		if err != nil {
			results = append(results, "ERROR:FileBakDir Create "+err.Error())
			return fmt.Errorf(strings.Join(results, "/n"))
		}
	}
	//备份原文件
	fileBak := backupDir + strings.Replace(fileSrc, "/", "_", -1)
	err = os.Rename(fileSrc, fileBak)
	if err != nil {
		cmd := exec.Command("mv", fileSrc, fileBak)
		_, err = cmd.Output()
		if err != nil {
			results = append(results, "ERROR:File Backup "+err.Error())
			return fmt.Errorf(strings.Join(results, "/n"))
		}
	}
	return err
}

func FileDownload(url string, file string, fileMod os.FileMode) error {
	var err error
	var res *http.Response
	res, err = http.Get(url)
	if err != nil {
		return fmt.Errorf("ERROR:File Download " + err.Error())
	}

	var outf *os.File
	outf, err = os.Create(file)
	defer outf.Close()
	if err != nil {
		return fmt.Errorf("ERROR:File Create " + err.Error())
	}
	io.Copy(outf, res.Body)
	outf.Chmod(fileMod)
	outf.Close()
	return err
}

func FileCreate(content string, file string) error {
	f, err := os.Create(file)
	defer f.Close()
	if err != nil {
		return fmt.Errorf("ERROR:FileMeta Create " + err.Error())
	}
	_, err = io.WriteString(f, content)
	if err != nil {
		return fmt.Errorf("ERROR:FileMeta Write " + err.Error())
	}
	f.Close()
	return err
}

func GetFileMod(file string) (os.FileMode, error) {
	var err error
	var info os.FileInfo

	info, err = os.Stat(file)
	if err != nil {
		return info.Mode(), fmt.Errorf("ERROR:File Stats " + err.Error())
	}

	return info.Mode(), err
}

func DropPageCache(filepath string) (err error) {
	handler, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer handler.Close()
	if err = unix.Fdatasync(int(handler.Fd())); err != nil {
		return err
	}
	if err = unix.Fadvise(int(handler.Fd()), 0, 0, unix.FADV_DONTNEED); err != nil {
		return err
	}
	return err
}

func GetFileMd5(file string) (string, error) {
	f, err := os.Open(file)
	defer f.Close()
	md5hash := md5.New()
	io.Copy(md5hash, f)
	return fmt.Sprintf("%x", md5hash.Sum(nil)), err
}

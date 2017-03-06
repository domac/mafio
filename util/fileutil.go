package util

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	WRITE_APPEND = 0
	WRITE_OVER   = 1
)

//生成channel code
func GenChannelCode() string {
	clock_str := time.Now().Format("20060102")
	h := md5.New()
	io.WriteString(h, clock_str)
	res := fmt.Sprintf("%x", h.Sum(nil))
	return res
}

//检查指定路径的文件是否存在
func CheckDataFileExist(filePath string) error {

	if filePath == "" {
		return errors.New("数据文件路径为空")
	}

	if _, err := os.Stat(filePath); err != nil {
		return errors.New("PathError:" + err.Error())
	}
	return nil
}

func IsExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

//逐行读
func ReadLine(fileName string) ([]string, error) {
	if err := CheckDataFileExist(fileName); err != nil {
		return []string{}, err
	}

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	result := []string{}

	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		if err != nil {
			if err == io.EOF {
				return result, nil
			}
			return nil, err
		}
		if line != "" && !strings.HasPrefix(line, "#") {
			result = append(result, line)
		}

	}
	return result, nil
}

//去重
func RemoveDuplicatesAndEmpty(a []string) (ret []string) {
	a_len := len(a)
	for i := 0; i < a_len; i++ {
		if (i > 0 && a[i-1] == a[i]) || len(a[i]) == 0 {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}

//删除文件
func RemoveFile(filepath string) error {
	err := os.Remove(filepath)
	if err != nil {
		return err
	}
	return nil
}

//根据路径创建文件
func CreateFile(filepath string) (string, error) {
	finfo, err := os.Stat(filepath)
	if err == nil {
		if finfo.IsDir() {
			return filepath, errors.New("filepath is a dir")
		} else {
			return filepath, errors.New("filepath exists")
		}
	}
	f, err := os.Create(filepath)
	if err != nil {
		fmt.Println("File path is not exist")
		return filepath, err
	}
	defer f.Close()
	return filepath, nil
}

//以追加的方式打开
func openToAppend(fpath string) (*os.File, error) {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return f, err
		}
	}
	return f, nil
}

//以覆盖的方式打开
func openToOverwrite(fpath string) (*os.File, error) {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return f, err
		}
	}
	return f, nil
}

//写入文件
func WriteIntoFile(filepath string, content []string, writeMode int) error {

	var f *os.File
	var err error

	if writeMode == WRITE_APPEND {
		f, err = openToAppend(filepath)
	} else {
		f, err = openToOverwrite(filepath)
	}

	if err != nil {
		return err
	}
	defer f.Close()

	for _, s := range content {
		fmt.Fprintln(f, s)
	}
	return nil
}

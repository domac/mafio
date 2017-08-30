package util

import (
	"os/exec"
	"strconv"
	"strings"
)

//直接指向命令脚本
func ShellRun(script string) error {
	return exec.Command("sh", "-c", script).Run()
}

//直接指向命令脚本，返回 []byte.
func ShellOutput(script string) ([]byte, error) {
	return exec.Command("sh", "-c", script).Output()
}

//直接指向命令脚本，返回 string.
func ShellString(script string) (result string, err error) {
	bs, err := ShellOutput(script)
	if err != nil {
		return
	}
	result = string(bs)
	return
}

//直接指向命令脚本，返回 int.
func ShellInt(script string) (result int, err error) {
	s, err := ShellString(script)
	if err != nil {
		return
	}
	s = strings.TrimSpace(s)
	result, err = strconv.Atoi(s)
	return
}

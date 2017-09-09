package util

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
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

//带超时的脚本实现
func ScriptRun(scripts []string, timeout int64) ([]byte, error) {
	var cmd *exec.Cmd
	var b bytes.Buffer

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
		cmd = exec.CommandContext(ctx, scripts[0], scripts[1:]...)
	} else {
		cmd = exec.Command(scripts[0], scripts[1:]...)
	}

	cmd.Stdout = &b
	cmd.Stderr = &b

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

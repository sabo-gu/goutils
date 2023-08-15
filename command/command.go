package command

import (
	"bufio"
	"bytes"
	"github.com/DoOR-Team/goutils/log"
	"io"
	"os"
	"os/exec"
	"time"
)

type FileWriterBuffer struct {
	File *os.File
}

func (buffer *FileWriterBuffer) Write(p []byte) (n int, err error) {
	writer := bufio.NewWriter(buffer.File)
	n, err = writer.Write(p)
	if err == nil {
		err = writer.Flush()
	}
	return n, err
}

var ShellToUse = "bash"

func Shellout(command string) (error, string) {
	var stdout bytes.Buffer
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	err := cmd.Run()
	return err, stdout.String()
}

func ShelloutWithBuffer(command string, out io.Writer, pid **exec.Cmd) error {
	*pid = exec.Command(ShellToUse, "-c", command)
	(*pid).Stdout = out
	(*pid).Stderr = out

	err := (*pid).Run()
	return err
}

func RunAsyncWithFile(command string, filePath string) error {

	file, _ := os.Create(filePath)
	log.Println(file.Name())
	defer file.Close()
	writer := &FileWriterBuffer{File: file}

	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = writer
	cmd.Stderr = writer
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}

func RunAsyncWithWriter(command string, writer *FileWriterBuffer) error {
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = writer
	cmd.Stderr = writer
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}

type RunStatus int

const (
	OK              RunStatus = 0
	ERROR           RunStatus = 1
	TimeExceedLimit RunStatus = 2
	UnKnown         RunStatus = 3
)

type TimeChecker struct {
	Ch          *chan RunStatus
	MaxTime     int
	Running     bool
	RunningTime int
}

func (tc *TimeChecker) timeExceed() {
	tc.RunningTime = 0
	for tc.Running {
		// log.Info(tc.RunningTime)
		if tc.RunningTime >= tc.MaxTime {
			*tc.Ch <- TimeExceedLimit
			break
		}
		time.Sleep(time.Second)
		tc.RunningTime++
		// log.Println("xxx", runningTime)
	}
}

func (tc *TimeChecker) Run() {
	go tc.timeExceed()
}

func run(command string, ch *chan RunStatus, out io.Writer, pid **exec.Cmd) {
	err := ShelloutWithBuffer(command, out, pid)
	if err != nil {
		log.Error(err)
		*ch <- ERROR
		return
	}
	*ch <- OK
}

func RunWithTimeLimit(command string, timeout int, verbose bool) (RunStatus, string) {

	var ch chan RunStatus
	ch = make(chan RunStatus, 2)
	tc := &TimeChecker{
		Ch: &ch,
		// 多给1s容器启动时间
		MaxTime: timeout,
		Running: true,
	}
	tc.Run()

	var output bytes.Buffer
	var pid *exec.Cmd

	if verbose {
		go run(command, &ch, os.Stdout, &pid)
	} else {
		go run(command, &ch, &output, &pid)
	}

	for {
		select {
		case v := <-ch:
			// log.Println(v, solverRunner.RunningTime)
			if pid != nil {
				pid.Process.Kill()
			}
			return v, output.String()
			// default:
			// 	// fmt.Println("get data timeout")
			// 	time.Sleep(time.Second)
		}
	}
}

package command

import (
	"fmt"
	"testing"

	"github.com/DoOR-Team/goutils/log"
)

func TestShellout(t *testing.T) {
	err, out := Shellout(`
ls -ltr
cd $GOPATH
pwd
whoami
`)
	if err != nil {
		log.Printf("error: %v\n", err)
	}
	fmt.Println("--- stdout ---")
	fmt.Println(out)
	fmt.Println("--- stderr ---")
}

func TestRunAsyncWithFile(t *testing.T) {
	RunAsyncWithFile(`
pwd
sleep 5
ls -la
`, "test.log")
}

func TestWithTimeout(t *testing.T) {

	status, output := RunWithTimeLimit(`
pwd
ls -la
sleep 5
ls -la
`, 10)
	log.Info(status)
	log.Info(output)

}

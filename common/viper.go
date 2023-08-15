package common

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

func LoadViperFromFiles(files ...string) error {
	viper.SetConfigType("yaml")
	for _, file := range files {
		// 读取配置文件
		if file == "" {
			log.Panic("配置文件不能为空")
		}
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		err = viper.MergeConfig(f)
		if err != nil {
			return err
		}
	}
	return nil
}

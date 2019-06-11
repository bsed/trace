// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/imdevlab/g"
	"github.com/bsed/trace/collector/misc"
	"github.com/bsed/trace/collector/service"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "collector",
	Short: "A brief description of your application",
	Long:  `接收器，存储并计算数据`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		misc.InitConfig("collector.yaml")
		g.InitLogger(misc.Conf.Common.LogLevel)
		g.L.Info("Application version", zap.String("version", misc.Conf.Common.Version))

		c := service.New(g.L)
		if err := c.Start(); err != nil {
			g.L.Fatal("collector start", zap.Error(err))
		}

		// 等待服务器停止信号
		chSig := make(chan os.Signal)
		signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)
		sig := <-chSig

		g.L.Info("collector received signal", zap.Any("signal", sig))
		c.Close()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.collector.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

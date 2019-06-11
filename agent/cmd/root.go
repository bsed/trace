package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/imdevlab/g"
	"github.com/bsed/trace/agent/misc"
	"github.com/bsed/trace/agent/service"
	"go.uber.org/zap"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "agent",
	Short: "采集服务和系统监控指标",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		misc.InitConfig("agent.yaml")
		g.InitLogger(misc.Conf.Common.LogLevel)
		g.L.Info("Application version", zap.String("version", misc.Conf.Common.Version))

		a := service.New(g.L)
		if err := a.Start(); err != nil {
			g.L.Fatal("agent start", zap.Error(err))
		}

		// 等待服务器停止信号
		chSig := make(chan os.Signal)
		signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)
		sig := <-chSig

		g.L.Info("agent received signal", zap.Any("signal", sig))
		a.Close()
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.agent.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

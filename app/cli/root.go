package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"Hamburger/app"
	"Hamburger/internal/config"
	"github.com/spf13/cobra"
)

const (
	defaultConfigPath = "config/config.json"
	pidFileName       = "hamburger.pid"
)

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	var configFile string

	rootCmd := &cobra.Command{
		Use:           "hamburger",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithConfig(configFile)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", defaultConfigPath, "config file")

	rootCmd.AddCommand(
		newGenerateCmd(),
		newRunCmd(&configFile),
		newTestCmd(&configFile),
		newReloadCmd(&configFile),
	)

	return rootCmd
}

func newGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "generate config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.CreateConfig()
		},
	}
}

func newRunCmd(configFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "run gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithConfig(*configFile)
		},
	}
}

func newTestCmd(configFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "test config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(*configFile)
			if err != nil {
				return err
			}
			_ = config.Merge(cfg)
			fmt.Fprintln(os.Stdout, "config ok")
			return nil
		},
	}
}

func newReloadCmd(configFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "reload service",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := readPid(pidFileName)
			if err != nil {
				return err
			}
			if err := signalProcess(pid); err != nil {
				return err
			}
			if err := waitPidFileRemoved(pidFileName, 30*time.Second); err != nil {
				return err
			}
			return startNewProcess(*configFile)
		},
	}
}

func runWithConfig(configFile string) error {
	hamburger, err := app.NewHamburgerApp(configFile)
	if err != nil {
		return err
	}
	hamburger.SetPidFile(pidFileName)
	if err := writePid(pidFileName); err != nil {
		return err
	}
	hamburger.Run()
	return nil
}

func writePid(path string) error {
	pid := strconv.Itoa(os.Getpid())
	return os.WriteFile(path, []byte(pid), 0644)
}

func readPid(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func signalProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

func waitPidFileRemoved(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for process shutdown")
}

func startNewProcess(configFile string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "run", "--config", configFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type systemInfo struct {
	version string
	kernel  string
	memory  string
	cpu     string
}

func printOSInfo() error {
	info, err := collectSystemInfo()
	if err != nil {
		return err
	}
	fmt.Printf("%s %s\n", commandText("version:"), info.version)
	fmt.Printf("%s %s\n", commandText("kernel:"), info.kernel)
	fmt.Printf("%s %s\n", commandText("memory:"), info.memory)
	fmt.Printf("%s %s\n", commandText("cpu:"), info.cpu)
	return nil
}

func collectSystemInfo() (systemInfo, error) {
	switch runtime.GOOS {
	case "windows":
		return collectWindowsSystemInfo()
	case "darwin":
		return collectDarwinSystemInfo()
	default:
		return collectLinuxSystemInfo()
	}
}

func collectWindowsSystemInfo() (systemInfo, error) {
	command := "$os=Get-CimInstance Win32_OperatingSystem; $cpu=(Get-CimInstance Win32_Processor | Select-Object -First 1).Name; $mem=[math]::Round($os.TotalVisibleMemorySize/1MB,2); Write-Output ($os.Caption + '|' + $os.Version + '|' + $os.BuildNumber + '|' + $cpu + '|' + $mem + ' GB')"
	text, err := runPowerShell(command)
	if err != nil {
		return systemInfo{}, err
	}
	parts := strings.SplitN(strings.TrimSpace(text), "|", 5)
	if len(parts) != 5 {
		return systemInfo{}, fmt.Errorf("osinfo parse failed")
	}
	return systemInfo{
		version: strings.TrimSpace(parts[0]) + " " + strings.TrimSpace(parts[1]),
		kernel:  "NT build " + strings.TrimSpace(parts[2]),
		cpu:     strings.TrimSpace(parts[3]),
		memory:  strings.TrimSpace(parts[4]),
	}, nil
}

func collectLinuxSystemInfo() (systemInfo, error) {
	version, _ := runSimpleCommand("sh", "-lc", "cat /etc/os-release | grep '^PRETTY_NAME=' | head -n1 | cut -d= -f2- | tr -d '\"'")
	if strings.TrimSpace(version) == "" {
		version, _ = runSimpleCommand("uname", "-s")
	}
	kernel, err := runSimpleCommand("uname", "-r")
	if err != nil {
		return systemInfo{}, err
	}
	cpu, _ := runSimpleCommand("sh", "-lc", "cat /proc/cpuinfo | grep 'model name' | head -n1 | cut -d: -f2-")
	memKBText, _ := runSimpleCommand("sh", "-lc", "cat /proc/meminfo | grep MemTotal | awk '{print $2}'")
	memory := strings.TrimSpace(memKBText) + " KB"
	if kb, parseErr := strconv.ParseFloat(strings.TrimSpace(memKBText), 64); parseErr == nil {
		memory = fmt.Sprintf("%.2f GB", kb/1024/1024)
	}
	return systemInfo{
		version: strings.TrimSpace(version),
		kernel:  strings.TrimSpace(kernel),
		cpu:     strings.TrimSpace(cpu),
		memory:  memory,
	}, nil
}

func collectDarwinSystemInfo() (systemInfo, error) {
	version, _ := runSimpleCommand("sw_vers", "-productVersion")
	kernel, err := runSimpleCommand("uname", "-r")
	if err != nil {
		return systemInfo{}, err
	}
	cpu, _ := runSimpleCommand("sysctl", "-n", "machdep.cpu.brand_string")
	memBytesText, _ := runSimpleCommand("sysctl", "-n", "hw.memsize")
	memory := strings.TrimSpace(memBytesText) + " B"
	if b, parseErr := strconv.ParseFloat(strings.TrimSpace(memBytesText), 64); parseErr == nil {
		memory = fmt.Sprintf("%.2f GB", b/1024/1024/1024)
	}
	return systemInfo{
		version: "macOS " + strings.TrimSpace(version),
		kernel:  strings.TrimSpace(kernel),
		cpu:     strings.TrimSpace(cpu),
		memory:  memory,
	}, nil
}

func runPowerShell(command string) (string, error) {
	encodedCommand := "[Console]::InputEncoding=[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new(); $OutputEncoding=[Console]::OutputEncoding; chcp 65001 | Out-Null; " + command
	return runSimpleCommand("powershell", "-NoProfile", "-Command", encodedCommand)
}

func runSimpleCommand(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", err
		}
		return "", fmt.Errorf("%s", text)
	}
	return text, nil
}


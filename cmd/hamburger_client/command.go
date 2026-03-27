package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

func (c *apiClient) execute(input string) (bool, error) {
	trimmed := strings.TrimSpace(input)
	trimmed = strings.TrimPrefix(trimmed, "\ufeff")
	if strings.HasPrefix(trimmed, "!") {
		return false, runShellCommand(strings.TrimSpace(strings.TrimPrefix(trimmed, "!")))
	}
	args := strings.Fields(trimmed)
	if len(args) == 0 {
		return false, nil
	}
	switch strings.ToLower(args[0]) {
	case "quit", "exit", "q":
		return true, nil
	case "h", "help":
		topic := ""
		if len(args) > 1 {
			topic = strings.ToLower(args[1])
		}
		return false, printHelp(topic)
	case "token":
		return false, c.handleToken(args)
	case "health":
		return false, c.requestAndPrint(http.MethodGet, "/api/health", nil, false)
	case "stat":
		return false, c.requestAndPrint(http.MethodGet, "/api/stat", nil, false)
	case "geo":
		return false, c.requestAndPrint(http.MethodGet, "/api/geo", nil, false)
	case "domain":
		return false, c.requestAndPrint(http.MethodGet, "/api/domain", nil, false)
	case "conn":
		return false, c.requestAndPrint(http.MethodGet, "/api/conn", nil, false)
	case "osinfo":
		return false, printOSInfo()
	case "login":
		return false, c.handleLogin(args)
	case "logout":
		return false, c.requestAndPrint(http.MethodPost, "/api/logout", nil, true)
	case "user":
		return false, c.handleUser(args)
	case "service":
		return false, c.handleService(args)
	case "server":
		return false, c.handleServer(args)
	default:
		return false, fmt.Errorf("unknown command: %s", args[0])
	}
}

func printHelp(topic string) error {
	switch topic {
	case "":
		fmt.Println("available top-level commands:")
		printHelpLine("h | help [topic]")
		printHelpLine("health")
		printHelpLine("stat")
		printHelpLine("geo")
		printHelpLine("domain")
		printHelpLine("conn")
		printHelpLine("osinfo")
		printHelpLine("login <username> <password>")
		printHelpLine("logout")
		printHelpLine("user")
		printHelpLine("service")
		printHelpLine("server")
		printHelpLine("token")
		printHelpLine("! <shell-command>")
		printHelpLine("quit | exit")
		fmt.Println("")
		fmt.Println("flags:")
		fmt.Printf("  %s %s\n", flagText("-host"), flagText("-port"))
		fmt.Println("")
		fmt.Println("use help <topic> to view subcommands, e.g. help user")
		return nil
	case "user":
		fmt.Println("user subcommands:")
		printHelpLine("user get")
		printHelpLine("user create <username> <password> [nickname] [avatar]")
		printHelpLine("user update <username> <password> [nickname] [avatar]")
		printHelpLine("user delete [username]")
		return nil
	case "service":
		fmt.Println("service subcommands:")
		printHelpLine("service start <domain>")
		printHelpLine("service stop <domain>")
		return nil
	case "server":
		fmt.Println("server subcommands:")
		printHelpLine("server stop <gateway|front|api|backend|latency|vpn|trojan|anytls>")
		printHelpLine("server restart <gateway|front|api|backend|latency|vpn|trojan|anytls>")
		return nil
	case "token":
		fmt.Println("token subcommands:")
		printHelpLine("token show")
		printHelpLine("token set <jwt>")
		printHelpLine("token clear")
		return nil
	default:
		return fmt.Errorf("unknown help topic: %s", topic)
	}
}

func runShellCommand(shellCommand string) error {
	if shellCommand == "" {
		return fmt.Errorf("usage: ! <shell-command>")
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		encodedCommand := "[Console]::InputEncoding=[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new(); $OutputEncoding=[Console]::OutputEncoding; chcp 65001 | Out-Null; " + shellCommand
		cmd = exec.Command("powershell", "-NoProfile", "-Command", encodedCommand)
	} else {
		cmd = exec.Command("sh", "-lc", shellCommand)
	}
	output, err := cmd.CombinedOutput()
	text := string(output)
	if err != nil {
		if strings.TrimSpace(text) != "" {
			fmt.Print(errorText(text))
		}
		if strings.TrimSpace(text) == "" || !strings.Contains(strings.ToLower(text), strings.ToLower(err.Error())) {
			fmt.Println(errorText(err.Error()))
		}
		return nil
	}
	if text != "" {
		fmt.Print(text)
	}
	return nil
}

func printHelpLine(line string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}
	first := parts[0]
	if strings.Contains(first, "|") {
		segs := strings.Split(first, "|")
		for i, seg := range segs {
			segs[i] = commandText(strings.TrimSpace(seg))
		}
		first = strings.Join(segs, " | ")
	} else {
		first = commandText(first)
	}
	fmt.Printf("  %s", first)
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, "-") || strings.HasPrefix(p, "<") || strings.HasPrefix(p, "[") {
			fmt.Printf(" %s", flagText(p))
			continue
		}
		fmt.Printf(" %s", commandText(p))
	}
	fmt.Println("")
}

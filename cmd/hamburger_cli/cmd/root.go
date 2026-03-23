package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	appgrpc "Hamburger/app/grpc"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type clientOptions struct {
	connType string
	addr     string
	timeout  time.Duration
	insecure bool
}

// Execute 执行CLI根命令
func Execute() error {
	return newRootCommand().Execute()
}

func newRootCommand() *cobra.Command {
	opts := &clientOptions{}
	root := &cobra.Command{
		Use:           "hamburger_cli",
		Short:         "Hamburger gRPC 控制工具",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&opts.connType, "type", "tcp", "gRPC连接类型: tcp 或 uds")
	root.PersistentFlags().StringVar(&opts.addr, "addr", "127.0.0.1:8081", "Hamburger gRPC地址")
	root.PersistentFlags().DurationVar(&opts.timeout, "timeout", 5*time.Second, "gRPC请求超时时间")
	root.PersistentFlags().BoolVar(&opts.insecure, "insecure", true, "是否使用非TLS连接")
	root.AddCommand(newStatusCommand(opts))
	root.AddCommand(newControlCommand(opts))
	return root
}

func newStatusCommand(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "查询服务状态信息",
	}
	cmd.AddCommand(
		newEmptyCallCommand(opts, "gateway", "查询Gateway状态", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.GetGatewayStatus(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "front", "查询FrontProxy状态", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.GetFrontProxyStatus(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "modifier", "查询Modifier状态", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.GetModifierManagerInfo(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "stat", "查询Stat配置", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.GetStatServerConfig(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "runtime", "查询运行时信息", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.GetRuntime(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "domain-map", "查询域名映射", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.GetDomainMap(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "domain-ports", "查询域名端口映射", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.GetDomainPorts(ctx, &appgrpc.Empty{})
		}),
	)
	return cmd
}

func newControlCommand(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "control",
		Short: "控制服务动作",
	}
	cmd.AddCommand(
		newReloadCommand(opts),
		newRestartCommand(opts),
		newDumpRuntimeCommand(opts),
	)
	return cmd
}

func newReloadCommand(opts *clientOptions) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "reload-config",
		Short: "重载配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithClient(opts, func(ctx context.Context, client appgrpc.AppServiceClient) error {
				resp, err := client.ReloadConfig(ctx, &appgrpc.ReloadConfigRequest{File: file})
				if err != nil {
					return err
				}
				return printProto(resp)
			})
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "配置文件路径，默认服务端配置")
	return cmd
}

func newRestartCommand(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "重启组件",
	}
	cmd.AddCommand(
		newEmptyCallCommand(opts, "gateway", "重启Gateway", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.ReStartGateway(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "front", "重启Front服务", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.ReStartFrontServer(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "api", "重启API服务", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.ReStartAPIServer(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "backend", "重启Backend服务", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.ReStartBackendServer(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "latency", "重启Latency服务", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.ReStartLatencyServer(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "vpn", "重启VPN服务", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.ReStartVPNServer(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "trojan", "重启Trojan服务", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.ReStartTrojanServer(ctx, &appgrpc.Empty{})
		}),
		newEmptyCallCommand(opts, "anytls", "重启AnyTLS服务", func(ctx context.Context, client appgrpc.AppServiceClient) (proto.Message, error) {
			return client.ReStartAnyTLSServer(ctx, &appgrpc.Empty{})
		}),
	)
	return cmd
}

func newDumpRuntimeCommand(opts *clientOptions) *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "dump-runtime",
		Short: "导出运行时快照",
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" {
				return fmt.Errorf("path 不能为空")
			}
			return runWithClient(opts, func(ctx context.Context, client appgrpc.AppServiceClient) error {
				resp, err := client.DumpRuntime(ctx, &appgrpc.DumpRuntimeRequest{Path: path})
				if err != nil {
					return err
				}
				return printProto(resp)
			})
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "运行时导出文件路径")
	return cmd
}

func newEmptyCallCommand(opts *clientOptions, use string, short string, call func(context.Context, appgrpc.AppServiceClient) (proto.Message, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithClient(opts, func(ctx context.Context, client appgrpc.AppServiceClient) error {
				resp, err := call(ctx, client)
				if err != nil {
					return err
				}
				return printProto(resp)
			})
		},
	}
}

func runWithClient(opts *clientOptions, fn func(context.Context, appgrpc.AppServiceClient) error) error {
	if opts.addr == "" {
		return fmt.Errorf("addr 不能为空")
	}
	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	conn, err := dial(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	client := appgrpc.NewAppServiceClient(conn)
	return fn(ctx, client)
}

func dial(ctx context.Context, opts *clientOptions) (*grpc.ClientConn, error) {
	connType := strings.ToLower(strings.TrimSpace(opts.connType))
	if connType == "" {
		connType = "tcp"
	}
	if connType != "tcp" && connType != "uds" {
		return nil, fmt.Errorf("不支持的连接类型: %s", opts.connType)
	}
	if connType == "uds" {
		if strings.TrimSpace(opts.addr) == "" {
			return nil, fmt.Errorf("uds 模式下 addr 不能为空")
		}
		return grpc.DialContext(
			ctx,
			"unix://"+opts.addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
				target := strings.TrimPrefix(addr, "unix://")
				return (&net.Dialer{}).DialContext(ctx, "unix", target)
			}),
			grpc.WithBlock(),
		)
	}
	var creds credentials.TransportCredentials
	if opts.insecure {
		creds = insecure.NewCredentials()
	} else {
		creds = credentials.NewTLS(&tls.Config{})
	}
	return grpc.DialContext(ctx, opts.addr, grpc.WithTransportCredentials(creds), grpc.WithBlock())
}

func printProto(msg proto.Message) error {
	data, err := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(data, '\n'))
	return err
}

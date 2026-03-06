package trojan

import (
	"context"
	"io"
	"net"
	"sync"

	"Hamburger/exp/trojan/common"
	"Hamburger/exp/trojan/core/tunnel"
	"Hamburger/exp/trojan/log"
)

type connSource interface {
	AcceptConn(nextTunnel tunnel.Tunnel) (tunnel.Conn, error)
	Close() error
}

type relayEngine struct {
	sources []connSource
	sink    tunnel.Client
	ctx     context.Context
	cancel  context.CancelFunc
}

var relayCopyBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

func newRelayEngine(ctx context.Context, cancel context.CancelFunc, sources []connSource, sink tunnel.Client) *relayEngine {
	return &relayEngine{
		sources: sources,
		sink:    sink,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (r *relayEngine) Run() error {
	r.relayConnLoop()
	<-r.ctx.Done()
	return nil
}

func (r *relayEngine) Close() error {
	r.cancel()
	_ = r.sink.Close()
	for _, source := range r.sources {
		_ = source.Close()
	}
	return nil
}

func (r *relayEngine) relayConnLoop() {
	for _, source := range r.sources {
		go func(source connSource) {
			for {
				inbound, err := source.AcceptConn(nil)
				if err != nil {
					select {
					case <-r.ctx.Done():
						return
					default:
					}
					log.Error(common.NewError("failed to accept connection").Base(err))
					continue
				}
				go r.handleConn(inbound)
			}
		}(source)
	}
}

func (r *relayEngine) handleConn(inbound tunnel.Conn) {
	defer inbound.Close()
	outbound, err := r.sink.DialConn(inbound.Metadata().Address, nil)
	if err != nil {
		log.Error(common.NewError("relay failed to dial connection").Base(err))
		return
	}
	defer outbound.Close()

	errChan := make(chan error, 2)
	go copyConn(errChan, inbound, outbound)
	go copyConn(errChan, outbound, inbound)

	select {
	case err = <-errChan:
		if err != nil {
			log.Error(err)
		}
	case <-r.ctx.Done():
	}
}

func copyConn(errChan chan error, a, b net.Conn) {
	buf := relayCopyBufferPool.Get().([]byte)
	defer relayCopyBufferPool.Put(buf)
	_, err := io.CopyBuffer(a, b, buf)
	errChan <- err
}

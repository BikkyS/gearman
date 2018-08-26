package gearman // import "github.com/nathanaelle/gearman"

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nathanaelle/gearman/v2/protocol"
)

type (
	Message struct {
		Reply  chan<- protocol.Packet
		Server Conn
		Pkt    protocol.Packet
	}
)

var (
	IOTimeout    time.Duration = 2 * time.Second
	RetryTimeout time.Duration = 50 * time.Millisecond
)

type pool struct {
	sync.Mutex
	pool     map[Conn]chan protocol.Packet
	msgQueue chan<- Message
	ctx      context.Context
	handlers map[string]int32
}

func (p *pool) newPool(ctx context.Context, msgQueue chan<- Message) {
	p.pool = make(map[Conn]chan protocol.Packet)
	p.handlers = make(map[string]int32)
	p.msgQueue = msgQueue
	p.ctx = ctx
}

func (p *pool) addServer(server Conn) error {
	p.Lock()

	if _, ok := p.pool[server]; ok {
		p.Unlock()
		return errors.New("server already exists: " + server.String())
	}

	pktchan := make(chan protocol.Packet, 100)
	p.pool[server] = pktchan
	p.Unlock()

	go p.wloop(server, pktchan)
	p.reconnect(server, pktchan)
	go p.rloop(server, pktchan)

	return nil
}

func (p *pool) listServers() []Conn {
	p.Lock()
	defer p.Unlock()

	list := make([]Conn, 0, len(p.pool))
	for k := range p.pool {
		list = append(list, k)
	}

	return list
}

func (p *pool) addHandler(h string) {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.handlers[h]; !ok {
		p.handlers[h] = 0
	}

	canDo := protocol.BuildPacket(protocol.CanDo, protocol.Opacify([]byte(h)))
	for _, server := range p.pool {
		server <- canDo
	}
}

func (p *pool) delHandler(h string) {
	p.Lock()
	defer p.Unlock()

	cantDo := protocol.BuildPacket(protocol.CantDo, protocol.Opacify([]byte(h)))
	for _, server := range p.pool {
		server <- cantDo
	}
}

func (p *pool) delAllHandlers() {
	p.Lock()
	defer p.Unlock()

	for h := range p.handlers {
		cantDo := protocol.BuildPacket(protocol.CantDo, protocol.Opacify([]byte(h)))
		for _, server := range p.pool {
			server <- cantDo
		}
	}
}

func (p *pool) sendTo(server Conn, pkt protocol.Packet) {
	p.Lock()
	defer p.Unlock()

	if c, ok := p.pool[server]; ok {
		c <- pkt
	}
}

func (p *pool) reconnect(server Conn, pktchan chan protocol.Packet) {
	p.Lock()
	defer p.Unlock()

	server.Redial()

	for h := range p.handlers {
		pktchan <- protocol.BuildPacket(protocol.CanDo, protocol.Opacify([]byte(h)))
	}

	p.msgQueue <- Message{pktchan, server, protocol.PktInternalEchoPacket}
}

func (p *pool) rloop(server Conn, pktchan chan protocol.Packet) {
	var err error
	var pkt protocol.Packet
	defer server.Close()

	pf := protocol.NewPacketFactory(server, 1<<20)

	for {
		select {
		case <-p.ctx.Done():
			return

		default:
			server.SetReadDeadline(time.Now().Add(IOTimeout))
			pkt, err = pf.Packet()

			switch {
			case err == nil:
				p.msgQueue <- Message{pktchan, server, pkt}

			case isTimeout(err):

			case isEOF(err):
				p.reconnect(server, pktchan)

			default:
				time.Sleep(IOTimeout)
				//				log.Println(err)
			}
		}
	}
}

func (p *pool) wloop(server Conn, sendTo chan protocol.Packet) {
	var err error
	defer server.Close()

	for {
		select {
		case <-p.ctx.Done():
			protocol.PktResetAbilities.WriteTo(server)
			return

		case data := <-sendTo:
			server.SetWriteDeadline(time.Now().Add(IOTimeout))
			_, err = data.WriteTo(server)

			for err != nil {
				//				log.Println(err)
				p.reconnect(server, sendTo)
				server.SetWriteDeadline(time.Now().Add(IOTimeout))
				_, err = data.WriteTo(server)
			}
		}
	}
}

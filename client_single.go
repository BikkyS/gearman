package	gearman

import	(
	"log"
)

type	(
	singleServer	struct {
		configured	bool
		pool
		jobs		map[TaskID]Task
		m_queue		chan message
		r_q		[]Task
	}
)


// create a new Client
// r_end is a channel to signal to the Client to end the process
func SingleServerClient(r_end <-chan struct{}, debug *log.Logger) Client {
	c		:= new(singleServer)
	c.m_queue	= make(chan message,10)
	c.jobs		= make(map[TaskID]Task)
	c.pool.new(c.m_queue, r_end)

	go client_loop(c,debug)

	return c
}


func (c *singleServer)MessageQueue() <-chan message {
	return c.m_queue
}

func (c *singleServer)EndSignal() <-chan struct{} {
	return c.r_end
}


//	Add a list of gearman server
func (c *singleServer)AddServers(servers ...Conn) Client {
	if c.configured || len(servers) == 0 {
		return	c
	}

	if len(servers) > 1 {
		servers = servers[0:1]
	}

	c.configured = true

	for _,server := range servers {
		c.add_server(server)
	}
	return	c
}


func (c *singleServer)Submit(req Task) Task {
	c.r_q	= append(c.r_q, req)

	for _,s := range c.list_servers() {
		c.send_to(s, req.Packet())
	}

	return	req
}


func (c *singleServer)AssignTask(tid TaskID) {
	c.jobs[tid]	= c.r_q[0]
	c.r_q		= c.r_q[1:]
}


func (c *singleServer)GetTask(tid TaskID) Task {
	if res,ok := c.jobs[tid]; ok {
		return	res
	}
	return	NilTask
}


func (c *singleServer)ExtractTask(tid TaskID) Task {
	if res,ok := c.jobs[tid]; ok {
		delete(c.jobs, tid)
		return	res
	}
	return	NilTask
}
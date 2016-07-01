package	gearman

import	(
	"log"
)

type	(
	Client	interface {
		AddServers(...Conn)	Client
		Submit(Task)		Task
		MessageQueue()		<-chan message
		AssignTask(tid TaskID)
		GetTask(TaskID)		Task
		ExtractTask(TaskID)	Task
		EndSignal()		<-chan struct{}
	}
)


func	client_loop(c Client, dbg *log.Logger) {
	mq	:= c.MessageQueue()
	end	:= c.EndSignal()

	for	{
		select	{
		case	msg := <-mq:
			debug(dbg, "CLI\t%s\n",msg.pkt)
			switch	msg.pkt.Cmd() {
			case	NOOP:

			case	ECHO_RES:
				debug(dbg, "CLI\tECHO [%s]\n",string(msg.pkt.At(0)))

			case	ERROR:
				debug(dbg, "CLI\tERR [%s] [%s]\n",msg.pkt.At(0),string(msg.pkt.At(1)))

			case	JOB_CREATED:
				tid,err	:= slice2TaskID(msg.pkt.At(0))
				if err != nil {
					panic(err)
				}
				c.AssignTask(tid)


			case	WORK_DATA, WORK_WARNING, WORK_STATUS:
				tid,err	:= slice2TaskID(msg.pkt.At(0))
				if err != nil {
					panic(err)
				}

				c.GetTask(tid).Handle(msg.pkt)

			case	WORK_COMPLETE, WORK_FAIL, WORK_EXCEPTION:
				tid,err	:= slice2TaskID(msg.pkt.At(0))
				if err != nil {
					panic(err)
				}

				c.ExtractTask(tid).Handle(msg.pkt)

			case	STATUS_RES:
				panic("status_res not wrote")

			case	OPTION_RES:
				panic("option_res not wrote")

			default:
				debug(dbg, "CLI\t%s\n", msg.pkt)
			}

		case	<-end:
			return
		}
	}
}

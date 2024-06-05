package kvsrv

// Put or Append
type PutAppendArgs struct {
	Key          string
	Value        string
	ClientID     int64
	LogicalCount int
}

type PutAppendReply struct {
	Value string
}

type GetArgs struct {
	Key          string
	ClientID     int64
	LogicalCount int
}

type GetReply struct {
	Value string
}

/* type ReportRecievedArgs struct {
	ClientID     int64
	LogicalCount int
}

type ReportRecieveReply struct {
} */

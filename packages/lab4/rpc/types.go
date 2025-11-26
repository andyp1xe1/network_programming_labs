package rpc

type SetArgs struct {
	ID    string
	Key   string
	Value string
}

type SetReply struct{}

type GetArgs struct {
	Key string
}

type GetReply struct {
	Value string
}

type DeleteArgs struct {
	ID  string
	Key string
}

type DeleteReply struct{}

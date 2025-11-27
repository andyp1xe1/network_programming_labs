package rpc

type SetArgs struct {
	ID      string
	Key     string
	Version int
	Value   string
}

type SetReply struct{}

type GetArgs struct {
	Key string
}

type GetReply struct {
	Value string
}

type DeleteArgs struct {
	ID      string
	Key     string
	Version int
}

type DeleteReply struct{}

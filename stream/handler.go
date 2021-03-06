package stream

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/tariel-x/stream/client"
)

var (
	ErrUnknownCmd   = errors.New("unknown cmd")
	ErrIncorrectCmd = errors.New("incorrect cmd")

	ResponseOK = "ok"

	availableCmds = map[string]struct{}{
		client.CmdPush:    {},
		client.CmdPull:    {},
		client.CmdGet:     {},
		client.CmdStatus:  {},
		client.CmdPrepare: {},
		client.CmdAccept:  {},
		client.CmdSet:     {},
	}
)

type ServerRequest interface {
	Message() string
	Address() string
	Name() string
}

type ServerResponse interface {
	Push(string)
}

type Log interface {
	Set(context.Context, int, string) error
	Get(context.Context, int) ([]string, error)
	Pull(context.Context, int) (chan string, error)
}

type AcceptMessage interface {
	N() int
	ID() string
	V() string
}

type Paxos interface {
	Commit(string) ([]AcceptMessage, error)
	Prepare(n int) (bool, AcceptMessage)
	Accept(n int, v, id string) bool
	Set(id string)
}

type Handler struct {
	paxos Paxos
	log   Log
}

func NewHandler(log Log, paxos Paxos) (*Handler, error) {
	return &Handler{
		log:   log,
		paxos: paxos,
	}, nil
}

type Request struct {
	ctx  context.Context
	cmd  string
	args []string
}

func (h *Handler) Process(ctx context.Context, message ServerRequest, response ServerResponse) error {
	parsed, err := parseRawMessage(message.Message())
	if err != nil {
		return err
	}
	parsed.ctx = ctx
	switch parsed.cmd {
	case client.CmdPush:
		request, err := NewPushRequest(*parsed)
		if err != nil {
			return err
		}
		return h.Push(request, response)
	case client.CmdGet:
		request, err := NewGetRequest(*parsed)
		if err != nil {
			return err
		}
		return h.Get(*request, response)
	case client.CmdPull:
		request, err := NewPullRequest(*parsed)
		if err != nil {
			return err
		}
		return h.Pull(*request, response)
	case client.CmdStatus:
		return h.Status(response)
	case client.CmdSet:
		request, err := NewSetRequest(*parsed)
		if err != nil {
			return err
		}
		return h.Set(request, response)
	case client.CmdPrepare:
		request, err := NewPrepareRequest(*parsed)
		if err != nil {
			return err
		}
		return h.Prepare(request, response)
	case client.CmdAccept:
		request, err := NewAcceptRequest(*parsed)
		if err != nil {
			return err
		}
		return h.Accept(request, response)
	default:
		return ErrUnknownCmd
	}
}

func parseRawMessage(message string) (*Request, error) {
	parsed := strings.SplitN(message, " ", 2)
	if len(parsed) == 0 {
		return nil, ErrIncorrectCmd
	}

	cmd, rawArgs := parsed[0], ""
	if len(parsed) == 2 {
		rawArgs = parsed[1]
	}

	if _, ok := availableCmds[cmd]; !ok {
		return nil, ErrIncorrectCmd
	}
	args := strings.Split(rawArgs, " ")
	return &Request{
		cmd:  cmd,
		args: args,
	}, nil
}

type GetRequest struct {
	Request
	n int
}

func NewGetRequest(request Request) (*GetRequest, error) {
	if request.cmd != client.CmdGet {
		return nil, ErrIncorrectCmd
	}
	if len(request.args) == 0 {
		return nil, ErrIncorrectCmd
	}
	n, err := strconv.Atoi(request.args[0])
	if err != nil {
		return nil, err
	}
	return &GetRequest{
		Request: request,
		n:       n,
	}, nil
}

type PullRequest struct {
	Request
	n int
}

func NewPullRequest(request Request) (*PullRequest, error) {
	if request.cmd != client.CmdPull {
		return nil, ErrIncorrectCmd
	}
	if len(request.args) == 0 {
		return nil, ErrIncorrectCmd
	}
	n, err := strconv.Atoi(request.args[0])
	if err != nil {
		return nil, err
	}
	return &PullRequest{
		Request: request,
		n:       n,
	}, nil
}

type PushRequest struct {
	Request
	v string
}

func NewPushRequest(request Request) (*PushRequest, error) {
	if request.cmd != client.CmdPush {
		return nil, ErrIncorrectCmd
	}
	if len(request.args) == 0 {
		return nil, ErrIncorrectCmd
	}
	return &PushRequest{
		Request: request,
		v:       request.args[0],
	}, nil
}

type PrepareRequest struct {
	Request
	n int
}

func NewPrepareRequest(request Request) (*PrepareRequest, error) {
	if request.cmd != client.CmdPrepare {
		return nil, ErrIncorrectCmd
	}
	if len(request.args) == 0 {
		return nil, ErrIncorrectCmd
	}
	n, err := strconv.Atoi(request.args[0])
	if err != nil {
		return nil, err
	}
	return &PrepareRequest{
		Request: request,
		n:       n,
	}, nil
}

type AcceptRequest struct {
	Request
	n  int
	id string
	v  string
}

func NewAcceptRequest(request Request) (*AcceptRequest, error) {
	if request.cmd != client.CmdAccept {
		return nil, ErrIncorrectCmd
	}
	if len(request.args) != 3 {
		return nil, ErrIncorrectCmd
	}
	n, err := strconv.Atoi(request.args[0])
	if err != nil {
		return nil, err
	}
	return &AcceptRequest{
		Request: request,
		n:       n,
		id:      request.args[1],
		v:       request.args[2],
	}, nil
}

type SetRequest struct {
	Request
	n  int
	id string
	v  string
}

func NewSetRequest(request Request) (*SetRequest, error) {
	if request.cmd != client.CmdSet {
		return nil, ErrIncorrectCmd
	}
	if len(request.args) != 3 {
		return nil, ErrIncorrectCmd
	}
	n, err := strconv.Atoi(request.args[0])
	if err != nil {
		return nil, err
	}
	return &SetRequest{
		Request: request,
		n:       n,
		id:      request.args[1],
		v:       request.args[2],
	}, nil
}

package cotcp

import (
	"errors"
)

var ErrNoReply = errors.New("reply to no-reply packet")

var ErrNoConnection = errors.New("no connection to host")

var ErrDeadlineExceed = deadlineExceedError{}

type deadlineExceedError struct{}

func (err deadlineExceedError) Error() string {
	return "deadline exceed"
}

func (err deadlineExceedError) Timeout() bool {
	return true
}

func (err deadlineExceedError) Temporary() bool {
	return true
}

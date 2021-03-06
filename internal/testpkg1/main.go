package testpkg1

import (
	"context"

	monkit "gopkg.in/spacemonkeygo/monkit.v2"
)

var (
	mon = monkit.Package()
)

func TestFunc(ctx context.Context, e error) (err error) {
	defer mon.Task()(&ctx)(&err)
	mon.Event("test_event")
	return e
}

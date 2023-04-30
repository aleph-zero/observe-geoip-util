package processors

import (
	"fmt"
	"github.com/gammazero/workerpool"
	"observe-geoip-util/options"
)

func ConsoleProcessor(payload string, _ options.Options, _ *workerpool.WorkerPool) {
	fmt.Println(payload)
}

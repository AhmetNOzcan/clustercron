package broker

import "fmt"

const (
	DefaultQueue = "clustercron:queue:default"
	QueuePrefix  = "clustercron:queue:"
)

func WorkerQueue(nodeID string) string {
	return fmt.Sprintf("%s%s", QueuePrefix, nodeID)
}

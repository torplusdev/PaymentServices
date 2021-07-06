package proxy

import (
	"sync"

	"paidpiper.com/payment-gateway/models"
)

type chainWrapper struct {
	ch          chan []byte
	commandType models.CommandType
}
type commandChannelStore struct {
	mutex          *sync.Mutex
	commandChannel map[string]chainWrapper
}

func NewCommandChainStore() *commandChannelStore {
	return &commandChannelStore{
		mutex:          &sync.Mutex{},
		commandChannel: make(map[string]chainWrapper),
	}
}

func (n *commandChannelStore) open(id string, commandType models.CommandType) <-chan []byte {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	ch := make(chan []byte, 2)

	n.commandChannel[id] = chainWrapper{
		ch:          ch,
		commandType: commandType,
	}

	return ch
}

func (n *commandChannelStore) close(id string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	ch, ok := n.commandChannel[id]
	if ok {
		delete(n.commandChannel, id)
		defer close(ch.ch)
	}

}

func (n *commandChannelStore) processResponse(commandId string, bs []byte) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	ch, ok := n.commandChannel[commandId]
	if ok {
		ch.ch <- bs
		return true
	}
	return false
}

package proxy

import (
	"sync"
)

type commandChannelStore struct {
	mutex          *sync.Mutex
	commandChannel map[string]chan []byte
}

func NewCommandChainStore() *commandChannelStore {
	return &commandChannelStore{
		mutex:          &sync.Mutex{},
		commandChannel: make(map[string]chan []byte),
	}
}

func (n *commandChannelStore) open(id string) <-chan []byte {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	ch := make(chan []byte, 2)
	n.commandChannel[id] = ch

	return ch
}

func (n *commandChannelStore) close(id string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	ch, ok := n.commandChannel[id]
	if ok {
		delete(n.commandChannel, id)
		defer close(ch)
	}

}

func (n *commandChannelStore) processResponse(commandId string, bs []byte) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	ch, ok := n.commandChannel[commandId]
	if ok {
		ch <- bs
		return true
	}
	return false
}

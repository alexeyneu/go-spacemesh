package hare

import (
	"github.com/gogo/protobuf/proto"
	"github.com/spacemeshos/go-spacemesh/hare/pb"
	"github.com/spacemeshos/go-spacemesh/p2p/service"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var Layer1 = &LayerId{Bytes32{1}}
var Layer2 = &LayerId{Bytes32{2}}
var Layer3 = &LayerId{Bytes32{3}}

func createMessage(t *testing.T, layer Byteable) []byte {
	hareMsg := &pb.HareMessage{}
	hareMsg.Message = &pb.InnerMessage{Layer: layer.Bytes()}
	serMsg, err := proto.Marshal(hareMsg)

	if err != nil {
		assert.Fail(t, "Failed to marshal data")
	}

	return serMsg
}

type MockInboxer struct {
	inbox chan *pb.HareMessage
}

func (inboxer *MockInboxer) createInbox(size uint32) chan *pb.HareMessage {
	inboxer.inbox = make(chan *pb.HareMessage, size)
	return inboxer.inbox
}

func TestBroker_Start(t *testing.T) {
	sim := service.NewSimulator()
	n1 := sim.NewNode()
	broker := NewBroker(n1)

	err := broker.Start()
	assert.Equal(t, nil, err)

	err = broker.Start()
	assert.Equal(t, "instance already started", err.Error())
}

// test that a message to a specific layer is delivered by the broker
func TestBroker_Received(t *testing.T) {
	sim := service.NewSimulator()
	n1 := sim.NewNode()
	n2 := sim.NewNode()

	broker := NewBroker(n1)
	broker.Start()

	inboxer := &MockInboxer{}
	broker.Register(Layer1, inboxer)

	serMsg := createMessage(t, Layer1)
	n2.Broadcast(ProtoName, serMsg)

	recv := <-inboxer.inbox

	assert.True(t, recv.Message.Layer[0] == Layer1.Bytes()[0])
}

// test that aborting the broker aborts
func TestBroker_Abort(t *testing.T) {
	sim := service.NewSimulator()
	n1 := sim.NewNode()

	broker := NewBroker(n1)
	broker.Start()

	timer := time.NewTimer(3 * time.Second)

	go broker.Close()

	select {
	case <-broker.CloseChannel():
		assert.True(t, true)
	case <-timer.C:
		assert.Fail(t, "timeout")
	}
}

func sendMessages(t *testing.T, layer *LayerId, n *service.Node, count int) {
	for i := 0; i < count; i++ {
		n.Broadcast(ProtoName, createMessage(t, layer))
	}
}

func waitForMessages(t *testing.T, inbox chan *pb.HareMessage, layer *LayerId, msgCount int) {
	for i := 0; i < msgCount; i++ {
		x := <-inbox
		assert.True(t, x.Message.Layer[0] == layer.Bytes()[0])
	}
}

// test flow for multiple layers
func TestBroker_MultipleLayers(t *testing.T) {
	sim := service.NewSimulator()
	n1 := sim.NewNode()
	n2 := sim.NewNode()
	const msgCount = 100

	broker := NewBroker(n1)
	broker.Start()

	inboxer1 := &MockInboxer{}
	inboxer2 := &MockInboxer{}
	inboxer3 := &MockInboxer{}
	broker.Register(Layer1, inboxer1)
	broker.Register(Layer2, inboxer2)
	broker.Register(Layer3, inboxer3)

	inbox1 := inboxer1.inbox
	inbox2 := inboxer2.inbox
	inbox3 := inboxer3.inbox

	go sendMessages(t, Layer1, n2, msgCount)
	go sendMessages(t, Layer2, n2, msgCount)
	go sendMessages(t, Layer3, n2, msgCount)

	waitForMessages(t, inbox1, Layer1, msgCount)
	waitForMessages(t, inbox2, Layer2, msgCount)
	waitForMessages(t, inbox3, Layer3, msgCount)

	assert.True(t, true)
}

func TestBroker_RegisterUnregister(t *testing.T) {
	sim := service.NewSimulator()
	n1 := sim.NewNode()
	broker := NewBroker(n1)
	broker.Start()
	inbox := &MockInboxer{}
	broker.Register(Layer1, inbox)
	assert.Equal(t, 1, len(broker.outbox))
	broker.Unregister(Layer1)
	assert.Equal(t, 0, len(broker.outbox))
}

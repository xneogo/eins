/*
 *  ┏┓      ┏┓
 *┏━┛┻━━━━━━┛┻┓
 *┃　　　━　　  ┃
 *┃   ┳┛ ┗┳   ┃
 *┃           ┃
 *┃     ┻     ┃
 *┗━━━┓     ┏━┛
 *　　 ┃　　　┃神兽保佑
 *　　 ┃　　　┃代码无BUG！
 *　　 ┃　　　┗━━━┓
 *　　 ┃         ┣┓
 *　　 ┃         ┏┛
 *　　 ┗━┓┓┏━━┳┓┏┛
 *　　   ┃┫┫  ┃┫┫
 *      ┗┻┛　 ┗┻┛
 @Time    : 2025/7/4 -- 17:52
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: onemq onemq/pulsar.go
*/

package onemq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/spf13/cast"
	"github.com/xneogo/eins/onelog"
	"github.com/xneogo/matrix/mmq"
	"time"
)

var (
	ErrPulsarBrokerNotNull           = errors.New("[one.mq.pulsar] broker cannot be null")
	ErrPulsarConfigNotNull           = errors.New("[one.mq.pulsar] config cannot be nil")
	ErrPulsarTopicInfoNotNull        = errors.New("[one.mq.pulsar] topic info cannot be null")
	ErrPulsarSubscriptionNameNotNull = errors.New("[one.mq.pulsar] subscription name cannot be null")
	ErrPulsarSubNameNotChinese       = errors.New("[one.mq.pulsar] subscription name cannot be chinese")
)

type PulsarHandler struct {
	msg      pulsar.Message
	consumer pulsar.Consumer
}

func (p *PulsarHandler) Ack(ctx context.Context) error {
	return p.consumer.Ack(p.msg)
}

func newPulsarHandler(consumer pulsar.Consumer, msg pulsar.Message) *PulsarHandler {
	return &PulsarHandler{
		msg:      msg,
		consumer: consumer,
	}
}

type PulsarConsumer struct {
	pulsar.Consumer
	Broker            string
	BrokerAdmin       string
	OperationTimeout  time.Duration
	ConnectionTimeout time.Duration

	client           pulsar.Client
	cfg              *pulsar.ConsumerOptions
	subscriptionName string
	url              string
	topic            string
}

func NewPulsarConsumer(broker, brokerAdmin, subscriptionName, topic string) (mmq.QReader, error) {

	if IsChinese(subscriptionName) {
		return nil, ErrPulsarSubNameNotChinese
	}
	if broker == "" {
		return nil, ErrPulsarBrokerNotNull
	}
	if subscriptionName == "" {
		return nil, ErrPulsarSubscriptionNameNotNull
	}
	if topic == "" {
		return nil, ErrPulsarTopicInfoNotNull
	}

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               broker,
		OperationTimeout:  OperationTimeout,
		ConnectionTimeout: ConnectionTimeout,
	})
	if err != nil {
		return nil, err
	}

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: subscriptionName,
		Type:             pulsar.Exclusive,
		MessageChannel:   make(chan pulsar.ConsumerMessage),
	})
	if err != nil {
		client.Close()
		return nil, err
	}

	return &PulsarConsumer{
		Consumer:          consumer,
		Broker:            broker,
		BrokerAdmin:       brokerAdmin,
		OperationTimeout:  OperationTimeout,
		ConnectionTimeout: ConnectionTimeout,
		client:            client,
		subscriptionName:  subscriptionName,
		url:               fmt.Sprintf("%s,%s,%s", broker, brokerAdmin, subscriptionName),
		topic:             topic,
	}, nil
}

func (p *PulsarConsumer) ReadMsgByGroup(ctx context.Context, topic, groupID string, value interface{}) (context.Context, error) {
	msg, err := p.Receive(ctx)
	if err != nil {
		return ctx, err
	}
	defer p.Ack(msg)

	err = json.Unmarshal(msg.Payload(), value)
	if err != nil {
		return ctx, err
	}
	return ctx, nil
}

// ReadMsgByPartition
// Todo: no msg by partition in pulsar
func (p *PulsarConsumer) ReadMsgByPartition(ctx context.Context, topic string, partition int, value interface{}) (context.Context, error) {
	return p.ReadMsgByGroup(ctx, topic, cast.ToString(partition), value)
}

func (p *PulsarConsumer) FetchMsgByGroup(ctx context.Context, topic, groupID string, value interface{}) (context.Context, mmq.AckHandler, error) {
	msg, err := p.Receive(ctx)
	if err != nil {
		return ctx, newPulsarHandler(p.Consumer, msg), err
	}

	err = json.Unmarshal(msg.Payload(), value)
	if err != nil {
		return ctx, newPulsarHandler(p.Consumer, msg), err
	}
	return ctx, newPulsarHandler(p.Consumer, msg), nil
}

func (p *PulsarConsumer) Close(ctx context.Context) error {
	onelog.Ctx(ctx).Debug().Msg("closing pulsar consumer")
	defer onelog.Ctx(ctx).Debug().Msg("closed pulsar consumer")

	p.Consumer.Close()
	p.client.Close()
	return nil
}

package rabbitmq

import (
	"b/big"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/streadway/amqp"
	"time"
)

type RabbitMq struct {
	UserName     string              //队列账户名
	PassWord     string              //队列密码
	Ip           string              //队列IP
	Port         int                 //队列端口
	Qos          int                 //并发数（准确的意思并不指并发，但常规可以这么理解）
	Heart        func(rmq *RabbitMq) //当队列掉线后触发该函数，函数中请自行重连及重新监听队列等操作
	conn         *amqp.Connection    //连接对象
	ch           *amqp.Channel       //连接管道
	isStartHeart bool                //心跳是否开启，用于掉线自动重连，本属性为隐藏属性，无需设置
	closeChan    chan *amqp.Error    //用于监听队列掉线的管道，本属性为隐藏属性，无需设置
}

/**
连接队列
返回：
	返回error对象，如果error对象值为nil表示成功，否则为失败
*/
func (p *RabbitMq) Connect() error {
	var err error
	if p.isStartHeart {
		return errors.New("请勿重复连接")
	}
	p.conn, err = amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", p.UserName, p.PassWord, p.Ip, p.Port))
	if err != nil {
		return err
	}
	p.ch, err = p.conn.Channel()
	if err != nil {
		return err
	}
	p.ch.Qos(p.Qos, 0, false)
	p.closeChan = make(chan *amqp.Error, 1)
	if !p.isStartHeart && p.Heart != nil {
		p.isStartHeart = true
		go func() {
			notifyClose := p.ch.NotifyClose(p.closeChan)
			for {
				select {
				case <-notifyClose:
					//队列已掉线
					p.Close()
					go p.Heart(p)
					return
				default:
					if !p.isStartHeart {
						go p.Heart(p)
						return
					}
					time.Sleep(5 * time.Second)
				}
			}
		}()
	}
	return err
}

/**
关闭连接
*/
func (p *RabbitMq) Close() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	//先关闭管道，在关闭连接，注意先后顺序
	p.isStartHeart = false
	close(p.closeChan)
	p.ch.Close()
	p.conn.Close()
}

/**
推送消息
传参：
	exchangeName：交换机名称
	routingKey：路由key
	body：消息主体
	headers：消息头
	properties：key只能填下列列举的key，value格式请百度，没有则不填。
				ContentType
				ContentEncoding
				Priority
				CorrelationId
				ReplyTo
				Expiration
				MessageId
				Timestamp
				Type
				UserId
				AppId
				ClusterId
返回：
	返回error对象，如果error对象值为nil表示成功，否则为失败
*/
func (p *RabbitMq) PushMsg(exchangeName string, routingKey string, body []byte, headers map[string]interface{}, properties map[string]interface{}) error {
	publishing := amqp.Publishing{
		Headers: headers,
		Body:    body,
	}
	for k, v := range properties {
		big.StuSetFieldVal(&publishing, k, v)
	}
	err := p.ch.Publish(exchangeName, routingKey, false, false, publishing)
	return err
}

/**
推送消息并等待响应
消费者应该从Properties属性中获取reply_to属性值，reply_to值为响应的监听队列，请将响应结果推送至该队列中
传参：
	exchangeName：交换机名称
	routingKey：路由key
	body：消息主体
	headers：消息头
	properties：key只能填下列列举的key，value格式请百度，没有则不填。
				ContentType
				ContentEncoding
				Priority
				CorrelationId
				ReplyTo
				Expiration
				MessageId
				Timestamp
				Type
				UserId
				AppId
				ClusterId
	timeout：等待响应超时时间，单位秒
返回：
	amqp.Delivery消息对象，如果有错误则会返回error对象
*/
func (p *RabbitMq) PushMsgAndWaitRes(exchangeName string, routingKey string, body []byte, headers map[string]interface{}, properties map[string]interface{}, timeout int) (amqp.Delivery, error) {
	publishing := amqp.Publishing{
		Headers: headers,
		Body:    body,
	}
	for k, v := range properties {
		big.StuSetFieldVal(&publishing, k, v)
	}
	var delivery amqp.Delivery
	//判断是否有应答队列，没有则添加
	uid := uuid.NewV4().String()
	if publishing.ReplyTo == "" {
		publishing.CorrelationId = uid
		publishing.ReplyTo = routingKey + "_reply_" + publishing.CorrelationId
	}
	//新建队列
	_, err := p.NewQueue(publishing.ReplyTo, false, true, false, true, nil)
	if err != nil {
		return delivery, err
	}
	//应答队列绑定交换机
	err = p.BindQueue(publishing.ReplyTo, publishing.ReplyTo, exchangeName, true, nil)
	if err != nil {
		return delivery, err
	}
	//推送消息
	err = p.ch.Publish(exchangeName, routingKey, false, false, publishing)
	if err != nil {
		return delivery, err
	}
	//监听应答消息
	ch, err := p.ch.Consume(publishing.ReplyTo, uid, true, false, false, false, nil)
	i := 0
	for {
		select {
		case delivery = <-ch:
			delivery.Ack(true)
			goto Loop
		default:
			i++
			if i > timeout {
				goto Loop
			}
			time.Sleep(1 * time.Second)
		}
	}
Loop:
	return delivery, nil
}

/**
监听消费消息（本注释中的消费者和监听者是一个意思）
传参：
	queueName：欲监听的队列名称
	consumer：监听者标识符，可随意填写，初始值为空字符串。
	autoAck：自动回执，如果为true，则收到消息后自动确认消息已处理。如果为false，则收到消息不自动确认消息处理情况。您需要手动调用 发送接受回执/发送拒绝回执 做消息手动回执。
	exclusive：是否独占，如果为true，则同时只允许当前监听者进行订阅，除此以外所有客户端都不能订阅此队列消息，直到客户端被关闭。
	noWait：是否等待消息ACK，为true则不等待，为false为等待。
	backcall：传入一个回调函数，当有新消息时调用该函数，函数定义格式：func demo(rmq *rabbitmq.RabbitMq, d amqp.Delivery){}
	args：其他参数，可以传nil
返回：
	返回error对象，如果error对象值为nil表示成功，否则为失败
*/
func (p *RabbitMq) ListenMsg(queueName string, consumer string, autoAck bool, exclusive bool, noWait bool, backcall func(rmq *RabbitMq, d amqp.Delivery), args map[string]interface{}) error {
	ch, err := p.ch.Consume(queueName, consumer, autoAck, exclusive, false, noWait, args)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case delivery := <-ch:
				if len(delivery.Body) == 0 {
					goto Loop
				} else {
					go backcall(p, delivery)
				}
			default:
				if p.isStartHeart == false {
					return
				}
				time.Sleep(1 * time.Second)
			}
		}
	Loop:
		p.Close()
	}()
	return err
}

/**
创建一个新的队列
传参：
	name：队列名称
	durable：是否持久化，持久化服务重启不会丢失数据，但会影响性能
	autoDelete：是否自动删除，如果为true，则没有任何客户端连接时，将自动删除队列。否则无论是否有客户端访问，队列和未处理的消息都将被永久保留
	exclusive：是否独占，如果为true，则队列只能被当前连接独占，连接断开则队列自动删除。否则队列将一直保持(持久化)，其它连接都可以访问此队列
	noWait：是否非阻塞
	args：其他参数，可以传nil
返回：
	队列对象，如果失败则会返回error错误对象
*/
func (p *RabbitMq) NewQueue(name string, durable bool, autoDelete bool, exclusive bool, noWait bool, args map[string]interface{}) (amqp.Queue, error) {
	return p.ch.QueueDeclare(name, durable, autoDelete, exclusive, noWait, args)
}

/**
删除队列
传参：
	name：队列名
	ifUnused：仅删除未使用的队列
	ifEmpty：仅删除空消息的队列
	noWait：不等待响应直接删除
返回：
	删除条件和错误信息
*/
func (p *RabbitMq) DelQueue(name string, ifUnused bool, ifEmpty bool, noWait bool) (int, error) {
	return p.ch.QueueDelete(name, ifUnused, ifEmpty, noWait)
}

/**
创建交换机
*/
func (p *RabbitMq) NewExchange(name, kind string, durable bool, autoDelete bool, internal bool, noWait bool, args map[string]interface{}) error {
	return p.ch.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args)
}

/**
删除交换机
*/
func (p *RabbitMq) DelExchange(name string, ifUnused bool, noWait bool) error {
	return p.ch.ExchangeDelete(name, ifUnused, noWait)
}

/**
绑定队列
*/
func (p *RabbitMq) BindQueue(name, key, exchange string, noWait bool, args map[string]interface{}) error {
	return p.ch.QueueBind(name, key, exchange, noWait, args)
}

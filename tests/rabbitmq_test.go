package tests

import (
	"b/rabbitmq"
	"github.com/streadway/amqp"
	"testing"
	"time"
)

func TestMq(t *testing.T) {
	println("开始连接MQ")
	rmq := rabbitmq.RabbitMq{
		UserName: "user",
		PassWord: "123456",
		Ip:       "192.168.0.80",
		Port:     5672,
		Qos:      1,
		Heart:    initAndReConn,
	}
	initAndReConn(&rmq)
	defer rmq.Close()
	//以下为除监听消息外的其他函数的用法
	//创建交换机
	//rmq.NewExchange("system.response", "topic", true, false, false, false,nil)
	//创建队列
	//_, err = rmq.NewQueue("Ys.Test.Queue", false, false, false, false, nil)
	//队列绑定交换机
	//rmq.BindQueue("Ys.Test.Queue","Ys.Test.Queue","system.response",false,nil)
	//推送消息
	//rmq.PushMsg("system.response", "Ys.Test.Queue", []byte("测试消息"), nil, nil)
	//推送消息并取响应结果
	//d, err := rmq.PushMsgAndWaitRes("system.response", "Ys.Test.Queue", []byte("测试消息"), nil, nil, 60)
	//	//if err != nil {
	//	//	println(err.Error())
	//	//	return
	//	//}
	//	//println(string(d.Body))
	time.Sleep(999 * time.Second) //长时间阻塞，防止主线程退出
}

/**
消息队列的连接和监听队列的代码请放于此函数中，
此函数可用于消息队列的首次连接及监听队列，
也可用于后续掉线重连自动触发本函数进行重新连接及监听队列。
传参：
	rmq：rabbitmq.RabbitMq结构体的指针
*/
func initAndReConn(rmq *rabbitmq.RabbitMq) {
	for {
		time.Sleep(5 * time.Second) //延迟5秒执行，以便腾出给原掉线的监听任务退出的时间
		err := rmq.Connect()
		if err == nil {
			//连接成功，跳出循环，执行后续监听代码
			break
		}
	}
	println("连接成功")
	//监听消息
	rmq.ListenMsg("测试队列", "broken", false, false, false, msgBackCall, nil)
}

//当有新消息时，自动触发该函数
func msgBackCall(rmq *rabbitmq.RabbitMq, d amqp.Delivery) {
	println(string(d.Body))
	d.Ack(true)
}

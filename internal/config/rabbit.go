package config

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"sysafari.com/customs/cguard/pkg/rabbit"
)

var (
	// A context that can be canceled to stop all consumers
	rabbitCtx    context.Context
	rabbitCancel context.CancelFunc
	// WaitGroup to track all running consumers
	consumerWg sync.WaitGroup
	// 确保初始化只执行一次
	initOnce sync.Once
)

// getRabbitConfig 从配置对象中构建RabbitMQ管理器配置
func getRabbitConfig() *rabbit.ManagerConfig {
	// 使用全局配置对象
	rabbitConfig := GlobalConfig.RabbitMQ

	managerConfig := &rabbit.ManagerConfig{
		// 连接配置
		URL:        rabbitConfig.URL,
		AutoAck:    true,
		AutoCreate: true,
	}

	return managerConfig
}

// InitRabbitMQ 初始化RabbitMQ客户端
func InitRabbitMQ() error {
	var err error

	// 使用sync.Once确保只初始化一次
	initOnce.Do(func() {
		// 创建可取消的上下文
		rabbitCtx, rabbitCancel = context.WithCancel(context.Background())

		// 初始化RabbitMQ管理器
		err = rabbit.InitializeWithConfig(getRabbitConfig())
		if err != nil {
			log.Errorf("Failed to initialize RabbitMQ manager: %v", err)
			return
		}

		log.Info("RabbitMQ manager initialized successfully ...")
	})

	return err
}

// CloseRabbitMQ 安全关闭所有RabbitMQ连接
func CloseRabbitMQ() {
	// 取消上下文，通知所有消费者停止
	if rabbitCancel != nil {
		rabbitCancel()
	}

	// 等待所有消费者完成
	consumerWg.Wait()

	// 关闭RabbitMQ管理器
	manager, err := rabbit.GetInstance()
	if err == nil && manager != nil {
		manager.Close()
	}

	log.Info("RabbitMQ connections closed")
}

// StartLwtRequestConsumer 启动LWT请求消费者
func StartLwtRequestConsumer(handleFuc func(msg string)) error {
	// 确保管理器已初始化
	if err := InitRabbitMQ(); err != nil {
		return err
	}

	// 获取管理器实例
	manager, err := rabbit.GetInstance()
	if err != nil {
		return err
	}

	// 获取底层客户端用于创建消费者
	client := manager.GetClient()

	// 获取队列配置
	rabbitConfig := GlobalConfig.RabbitMQ
	exchangeName := rabbitConfig.Exchange
	exchangeType := rabbitConfig.ExchangeType
	if exchangeType == "" {
		exchangeType = "direct"
	}
	log.Infof("Declaring exchange: %s, type: %s", exchangeName, exchangeType)

	// 声明交换机
	err = manager.DeclareExchange(rabbit.ExchangeConfig{
		Name:       exchangeName,
		Type:       exchangeType,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
	})
	if err != nil {
		log.Errorf("Failed to declare exchange: %v", err)
		return err
	}

	// 获取队列名称
	queueName := rabbitConfig.Queue.LwtReq
	if queueName == "" {
		return errors.New("import xml queue name is empty")
	}

	// 声明队列
	err = manager.DeclareQueue(rabbit.QueueConfig{
		Name:       queueName,
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
	})
	if err != nil {
		return errors.New("failed to declare queue: " + err.Error())
	}

	// 绑定队列到交换机
	err = manager.BindQueue(rabbit.BindingConfig{
		QueueName:    queueName,
		ExchangeName: exchangeName,
		RoutingKey:   "",
	})
	if err != nil {
		return errors.New("failed to bind queue: " + err.Error())
	}

	// 创建消费者
	consumerName := "cguard_lwt_request_consumer"
	consumer, err := client.NewConsumer(queueName, consumerName)
	if err != nil {
		return err
	}

	// 跟踪此消费者
	consumerWg.Add(1)

	// 在goroutine中启动消费
	go func() {
		defer consumerWg.Done()
		log.Infof("consumer %s started ...", consumerName)

		// 定义消息处理函数 - 根据 ConsumeWithContext 接口定义匹配参数类型
		messageHandler := func(ctx context.Context, msg []byte) error {
			log.Infof("Received LWT request message")

			// 处理消息
			if handleFuc != nil {
				handleFuc(string(msg))
			}
			// 消息处理成功
			return nil
		}

		// 开始消费
		consumer.ConsumeWithContext(rabbitCtx, messageHandler)
		log.Infof("consumer %s finished consuming a message ...", consumerName)
	}()

	return nil
}

// PublishMessage 发布消息到RabbitMQ
func PublishMessage(message string, queueName string) error {
	// 确保管理器已初始化
	if err := InitRabbitMQ(); err != nil {
		return err
	}

	// 获取管理器实例
	manager, err := rabbit.GetInstance()
	if err != nil {
		return err
	}

	// 获取配置信息
	rabbitConfig := GlobalConfig.RabbitMQ
	exchangeName := rabbitConfig.Exchange

	// 确保队列存在
	if queueName == "" {
		return errors.New("queue name is empty")
	}

	// 确保队列已声明并绑定到交换机
	err = manager.DeclareQueue(rabbit.QueueConfig{
		Name:       queueName,
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
	})
	if err != nil {
		return errors.New("failed to declare queue: " + err.Error())
	}

	// 绑定队列到交换机（如果还没有绑定）
	err = manager.BindQueue(rabbit.BindingConfig{
		QueueName:    queueName,
		ExchangeName: exchangeName,
		RoutingKey:   queueName, // 使用队列名作为路由键
	})
	if err != nil {
		return errors.New("failed to bind queue: " + err.Error())
	}

	// 发布消息，使用队列名作为路由键
	err = manager.PublishMessage(exchangeName, queueName, message)
	if err != nil {
		log.Errorf("Failed to publish message: %v", err)
		return err
	}

	log.Infof("Message published to exchange '%s' with queue '%s'", exchangeName, queueName)
	return nil
}

// PublishMessageWithContext 使用上下文发布消息到RabbitMQ（支持超时和取消）
func PublishMessageWithContext(ctx context.Context, message string, queueName string) error {
	// 确保管理器已初始化
	if err := InitRabbitMQ(); err != nil {
		return err
	}

	// 获取管理器实例
	manager, err := rabbit.GetInstance()
	if err != nil {
		return err
	}

	// 获取底层客户端用于发布消息
	client := manager.GetClient()

	// 获取配置信息
	rabbitConfig := GlobalConfig.RabbitMQ
	exchangeName := rabbitConfig.Exchange

	// 确保队列存在
	if queueName == "" {
		return errors.New("queue name is empty")
	}

	// 确保队列已声明并绑定到交换机
	err = manager.DeclareQueue(rabbit.QueueConfig{
		Name:       queueName,
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
	})
	if err != nil {
		return errors.New("failed to declare queue: " + err.Error())
	}

	// 绑定队列到交换机（如果还没有绑定）
	err = manager.BindQueue(rabbit.BindingConfig{
		QueueName:    queueName,
		ExchangeName: exchangeName,
		RoutingKey:   queueName, // 使用队列名作为路由键
	})
	if err != nil {
		return errors.New("failed to bind queue: " + err.Error())
	}

	// 发布消息（带上下文），使用队列名作为路由键
	err = client.Publish(
		ctx,
		exchangeName,
		queueName, // 使用队列名作为路由键
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(message),
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		log.Errorf("Failed to publish message with context: %v", err)
		return err
	}

	log.Infof("Message published to exchange '%s' with queue '%s'", exchangeName, queueName)
	return nil
}

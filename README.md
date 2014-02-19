# RabbitMQ to Resque workers

This project is still under development. Mome work needs to be done in order to be production ready.

It is written in GO to allow high concurrency with very little overhead, but it is designed to be used in a Ruby application.

rabbitmq-to-resque listens to specific messages on RabitMQ and pushes the messages into Resque/Sidekiq workers.

## Configuration

The service uses a JSON configuration file to 

```json
{
	"rabbitmq": {
		"url": "amqp://guest:guest@localhost:5670/"
	},
	"redis": {
		"host": "localhost",
		"port": "6379"
	},
	"mappings": [
		{
			"exchange": "awesome_application",
			"queue": "products.update",
			"routingKey": "products.update",
			"workerQueue": "default",
			"workerClass": "DataFeedImportWorker"
		}
	]
}
```

## Usage

Go is required for compilation. `make all` will compile into the bin directory.

Start the server:
```bash
./bin/rabbitmq-to-resque config.json
```

## High Availability

It will retry reconnecting to RabbitMQ when the connection is lost. When using mirrored queues it will switch to an available server. I currently use HAProxy to load balance between a cluster of RabitMQ instances.


#### HAProxy configuration
This is a basic configuration that will load balance between available rabbit servers that live on the save machine. In production each instance should be in a different server.
```haproxy
global
    log 127.0.0.1    local0 info
    maxconn 4096
    stats socket /tmp/haproxy.socket uid haproxy mode 770 level admin
    daemon
defaults
    log    global
    mode    tcp
    option    tcplog
    option    dontlognull
    retries    3
    option redispatch
    maxconn    2000
    timeout connect    5s
    timeout client 120s
    timeout server 120s
listen rabbitmq_local_cluster 127.0.0.1:5670
    mode tcp
    balance roundrobin
    server rabbit 127.0.0.1:5672 check inter 5000 rise 2 fall 3
    server rabbit_1 127.0.0.1:5673 check inter 5000 rise 2 fall 3
    server rabbit_2 127.0.0.1:5674 check inter 5000 rise 2 fall 3
listen private_monitoring :8100
    mode http
    option httplog
    stats enable
    stats uri   /stats
    stats refresh 5s
```

## Missing
- Handle the case when all RabbitMQ instances are down.
- Pipelined writes to Redis.
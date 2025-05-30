# kafka exporter

Simple consumer-producer pair.
Consumer consumes messages from the source topic, while producer produces them into the destination topic.

## Run locally

1. Launch Kafka
```bash
docker-compose up
```
2. Run code
```bash
go run main.go
```

## Useful commands:

```bash
# list topics
kafka-topics --bootstrap-server localhost:9092 --list

# create source and destination topics
kafka-topics --bootstrap-server localhost:9092 --create --topic source-topic --partitions 1 --replication-factor 1
kafka-topics --bootstrap-server localhost:9092 --create --topic destination-topic --partitions 1 --replication-factor 1

# produce messages
kafka-console-producer --bootstrap-server localhost:9092 --topic my-topic --property "parse.key=true" --property "key.separator=:"

# consume messages
kafka-console-consumer --bootstrap-server localhost:9092 --topic destination-topic --from-beginning --property print.key=true --property key.separator=:
```

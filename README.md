# kafka exporter

Simple consumer-producer pair.
Consumer consumes messages from the source topic, while producer produces them into the destination topic.

## Run locally

1. Generate certificates
```bash
bash scripts/cert-generator.sh
```
2. Launch Kafka using docker compose
```bash
docker-compose up
```
3. Run kafka exporter
```bash
go run cmd/kafka-exporter/main.go
```

## Useful commands:

Login into the kafka container:
```bash
docker exec -it broker bash
```

Now you can execute the next commands:

```bash
# list topics
kafka-topics --bootstrap-server localhost:9092 --list

# create source and destination topics
kafka-topics --bootstrap-server localhost:9092 --create --topic source-topic --partitions 1 --replication-factor 1
kafka-topics --bootstrap-server localhost:9092 --create --topic destination-topic --partitions 1 --replication-factor 1

# produce messages
kafka-console-producer --bootstrap-server localhost:9092 --topic source-topic --property parse.key=true --property key.separator=:

# consume messages
kafka-console-consumer --bootstrap-server localhost:9092 --topic destination-topic --from-beginning --property print.key=true --property key.separator=:
```

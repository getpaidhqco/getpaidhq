# Minimal Kafka image for production deployments
# This is designed for low-traffic environments and minimal resource usage

FROM alpine:3.19

# Install required packages
RUN apk add --no-cache \
    openjdk11-jre-headless \
    bash \
    curl \
    && rm -rf /var/cache/apk/*

# Set environment variables
ENV KAFKA_VERSION=2.13-3.5.1
ENV KAFKA_HOME=/opt/kafka
ENV JAVA_HOME=/usr/lib/jvm/java-11-openjdk

# Create kafka user and directory
RUN addgroup -g 1001 kafka && \
    adduser -D -s /bin/bash -u 1001 -G kafka kafka

# Download and install Kafka
RUN mkdir -p ${KAFKA_HOME} && \
    curl -fsSL https://archive.apache.org/dist/kafka/3.5.1/kafka_${KAFKA_VERSION}.tgz | \
    tar -xz --strip-components=1 -C ${KAFKA_HOME} && \
    chown -R kafka:kafka ${KAFKA_HOME}

# Create necessary directories
RUN mkdir -p /opt/kafka/logs /opt/kafka/config && \
    chown -R kafka:kafka /opt/kafka/logs /opt/kafka/config

# Copy minimal configuration files
COPY <<EOF /opt/kafka/config/server.properties
# Minimal Kafka configuration for low-traffic production
broker.id=0
listeners=PLAINTEXT://0.0.0.0:9092
advertised.listeners=PLAINTEXT://kafka:9092
log.dirs=/opt/kafka/logs
num.network.threads=3
num.io.threads=8
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600
num.partitions=1
num.recovery.threads.per.data.dir=1
offsets.topic.replication.factor=1
transaction.state.log.replication.factor=1
transaction.state.log.min.isr=1
log.retention.hours=168
log.retention.bytes=1073741824
log.segment.bytes=1073741824
log.retention.check.interval.ms=300000
log.cleanup.policy=delete
auto.create.topics.enable=true
delete.topic.enable=true
group.initial.rebalance.delay.ms=0
# Optimize for low resource usage
num.replica.fetchers=1
replica.fetch.max.bytes=1048576
replica.fetch.min.bytes=1
replica.fetch.wait.max.ms=500
replica.high.watermark.checkpoint.interval.ms=5000
replica.socket.timeout.ms=30000
replica.socket.receive.buffer.bytes=65536
replica.lag.time.max.ms=10000
controller.socket.timeout.ms=30000
controller.message.queue.size=10
# Memory optimization
log.flush.interval.messages=10000
log.flush.interval.ms=1000
# Compression for space efficiency
compression.type=lz4
EOF

# Copy ZooKeeper configuration (standalone mode)
COPY <<EOF /opt/kafka/config/zookeeper.properties
# Minimal ZooKeeper configuration
dataDir=/opt/kafka/zookeeper
clientPort=2181
maxClientCnxns=0
admin.enableServer=false
# Optimize for low resource usage
tickTime=2000
initLimit=10
syncLimit=5
autopurge.snapRetainCount=3
autopurge.purgeInterval=24
EOF

# Set proper permissions
RUN chown -R kafka:kafka /opt/kafka/config

# Create startup script
COPY <<EOF /opt/kafka/start-kafka.sh
#!/bin/bash
set -e

# Start ZooKeeper in background
echo "Starting ZooKeeper..."
\$KAFKA_HOME/bin/zookeeper-server-start.sh \$KAFKA_HOME/config/zookeeper.properties &

# Wait for ZooKeeper to start
echo "Waiting for ZooKeeper to start..."
until \$KAFKA_HOME/bin/zookeeper-shell.sh localhost:2181 <<< "ls /" > /dev/null 2>&1; do
  sleep 1
done

echo "ZooKeeper started successfully"

# Start Kafka
echo "Starting Kafka..."
exec \$KAFKA_HOME/bin/kafka-server-start.sh \$KAFKA_HOME/config/server.properties
EOF

RUN chmod +x /opt/kafka/start-kafka.sh && \
    chown kafka:kafka /opt/kafka/start-kafka.sh

# Create health check script
COPY <<EOF /opt/kafka/health-check.sh
#!/bin/bash
# Simple health check for Kafka
\$KAFKA_HOME/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092 > /dev/null 2>&1
exit \$?
EOF

RUN chmod +x /opt/kafka/health-check.sh && \
    chown kafka:kafka /opt/kafka/health-check.sh

# Switch to kafka user
USER kafka

# Set working directory
WORKDIR /opt/kafka

# Expose ports
EXPOSE 9092 2181

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD ["/opt/kafka/health-check.sh"]

# Start Kafka
CMD ["/opt/kafka/start-kafka.sh"]
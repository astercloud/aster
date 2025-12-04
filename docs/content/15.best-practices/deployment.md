---
title: 部署实践
description: Agent 应用的容器化部署和运维策略
navigation:
  icon: i-lucide-rocket
---

# 部署最佳实践

本章介绍 Agent SDK 应用的生产部署策略和运维实践。

## 🎯 部署架构

```mermaid
graph TB
    subgraph "Load Balancer"
        LB[Nginx/ALB]
    end

    subgraph "Application Layer"
        APP1[Agent Service 1]
        APP2[Agent Service 2]
        APP3[Agent Service 3]
    end

    subgraph "Storage Layer"
        REDIS[(Redis Cache)]
        PG[(PostgreSQL)]
        S3[(Object Storage)]
    end

    subgraph "Observability"
        PROM[Prometheus]
        GRAF[Grafana]
        LOKI[Loki]
    end

    LB --> APP1
    LB --> APP2
    LB --> APP3

    APP1 --> REDIS
    APP1 --> PG
    APP1 --> S3

    APP2 --> REDIS
    APP2 --> PG
    APP2 --> S3

    APP3 --> REDIS
    APP3 --> PG
    APP3 --> S3

    APP1 -.-> PROM
    APP2 -.-> PROM
    APP3 -.-> PROM

    PROM --> GRAF
    APP1 -.-> LOKI
    APP2 -.-> LOKI
    APP3 -.-> LOKI
```

## 🐳 容器化部署

### Dockerfile

```dockerfile
# 多阶段构建
FROM golang:1.21-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -X main.version=$(git describe --tags --always)" \
    -o /app/agent-service ./cmd/server

# 运行阶段
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 创建非 root 用户
RUN addgroup -g 1000 agent && \
    adduser -D -u 1000 -G agent agent

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/agent-service .

# 创建数据目录
RUN mkdir -p /app/data /app/logs && \
    chown -R agent:agent /app

# 切换到非 root 用户
USER agent

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 暴露端口
EXPOSE 8080 9090

# 启动命令
CMD ["./agent-service"]
```

### Docker Compose (开发环境)

```yaml
# docker-compose.yml
version: "3.8"

services:
  agent-service:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - LOG_LEVEL=info
      - REDIS_URL=redis://redis:6379
      - POSTGRES_URL=postgres://agent:password@postgres:5432/agentdb
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
    depends_on:
      - redis
      - postgres
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_DB=agentdb
      - POSTGRES_USER=agent
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres-data:/var/lib/postgresql/data
    restart: unless-stopped

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
      - "--storage.tsdb.path=/prometheus"
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana
    restart: unless-stopped

volumes:
  redis-data:
  postgres-data:
  prometheus-data:
  grafana-data:
```

### 构建和运行

```bash
# 构建镜像
docker build -t agent-service:latest .

# 运行容器
docker run -d \
  --name agent-service \
  -p 8080:8080 \
  -p 9090:9090 \
  -e ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY} \
  -v $(pwd)/data:/app/data \
  agent-service:latest

# 使用 Docker Compose
docker-compose up -d

# 查看日志
docker-compose logs -f agent-service

# 停止服务
docker-compose down
```

## ☸️ Kubernetes 部署

### Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-service
  namespace: production
  labels:
    app: agent-service
    version: v1.0.0
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: agent-service
  template:
    metadata:
      labels:
        app: agent-service
        version: v1.0.0
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      # 安全上下文
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000

      # 服务账户
      serviceAccountName: agent-service

      # 初始化容器
      initContainers:
        - name: wait-for-redis
          image: busybox:latest
          command:
            - sh
            - -c
            - |
              until nc -z redis 6379; do
                echo "Waiting for Redis..."
                sleep 2
              done

      containers:
        - name: agent-service
          image: your-registry/agent-service:v1.0.0
          imagePullPolicy: Always

          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
            - name: metrics
              containerPort: 9090
              protocol: TCP

          # 环境变量
          env:
            - name: LOG_LEVEL
              value: "info"
            - name: ANTHROPIC_API_KEY
              valueFrom:
                secretKeyRef:
                  name: agent-secrets
                  key: anthropic-api-key
            - name: REDIS_URL
              value: "redis://redis:6379"
            - name: POSTGRES_URL
              valueFrom:
                secretKeyRef:
                  name: agent-secrets
                  key: postgres-url

          # 资源限制
          resources:
            requests:
              cpu: 500m
              memory: 512Mi
            limits:
              cpu: 2000m
              memory: 2Gi

          # 健康检查
          livenessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 3
            failureThreshold: 3

          readinessProbe:
            httpGet:
              path: /ready
              port: http
            initialDelaySeconds: 10
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3

          # 启动探针
          startupProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 0
            periodSeconds: 10
            timeoutSeconds: 3
            failureThreshold: 30

          # 挂载卷
          volumeMounts:
            - name: data
              mountPath: /app/data
            - name: logs
              mountPath: /app/logs
            - name: config
              mountPath: /app/config
              readOnly: true

      # 卷定义
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: agent-data-pvc
        - name: logs
          emptyDir: {}
        - name: config
          configMap:
            name: agent-config

      # 节点亲和性
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app
                      operator: In
                      values:
                        - agent-service
                topologyKey: kubernetes.io/hostname
```

### Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: agent-service
  namespace: production
  labels:
    app: agent-service
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 80
      targetPort: http
      protocol: TCP
    - name: metrics
      port: 9090
      targetPort: metrics
      protocol: TCP
  selector:
    app: agent-service
```

### Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: agent-service
  namespace: production
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - api.example.com
      secretName: agent-tls
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: agent-service
                port:
                  name: http
```

### ConfigMap 和 Secret

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: agent-config
  namespace: production
data:
  config.yaml: |
    server:
      port: 8080
      metrics_port: 9090
    agent:
      max_agents: 100
      token_limit: 150000
    logging:
      level: info
      format: json

---
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: agent-secrets
  namespace: production
type: Opaque
stringData:
  anthropic-api-key: "sk-ant-xxxxx"
  postgres-url: "postgres://user:pass@postgres:5432/db"
```

### HorizontalPodAutoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: agent-service-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: agent-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
    - type: Pods
      pods:
        metric:
          name: agent_active_count
        target:
          type: AverageValue
          averageValue: "50"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 30
        - type: Pods
          value: 2
          periodSeconds: 30
      selectPolicy: Max
```

### 部署命令

```bash
# 创建命名空间
kubectl create namespace production

# 应用配置
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml

# 部署应用
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml
kubectl apply -f hpa.yaml

# 查看状态
kubectl get all -n production
kubectl get pods -n production -w

# 查看日志
kubectl logs -f -n production -l app=agent-service

# 扩缩容
kubectl scale deployment agent-service -n production --replicas=5

# 滚动更新
kubectl set image deployment/agent-service \
  agent-service=your-registry/agent-service:v1.1.0 \
  -n production

# 查看滚动状态
kubectl rollout status deployment/agent-service -n production

# 回滚
kubectl rollout undo deployment/agent-service -n production
```

## 🔄 CI/CD 集成

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Build and Deploy

on:
  push:
    branches: [main]
    tags: ["v*"]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: "1.21"

      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

      - name: Check coverage
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
          echo "Coverage: $coverage%"
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "Coverage below 80%"
            exit 1
          fi

  build:
    needs: test
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2

      - name: Log in to Container Registry
        uses: docker/login-action@v2
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v4
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=branch
            type=ref,event=pr
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=sha

      - name: Build and push Docker image
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy-staging:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    environment:
      name: staging
      url: https://staging.example.com

    steps:
      - uses: actions/checkout@v3

      - name: Set up kubectl
        uses: azure/setup-kubectl@v3

      - name: Configure kubectl
        run: |
          echo "${{ secrets.KUBE_CONFIG_STAGING }}" | base64 -d > kubeconfig
          export KUBECONFIG=kubeconfig

      - name: Deploy to staging
        run: |
          kubectl set image deployment/agent-service \
            agent-service=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:sha-${GITHUB_SHA::7} \
            -n staging
          kubectl rollout status deployment/agent-service -n staging

      - name: Run smoke tests
        run: |
          ./scripts/smoke-test.sh https://staging.example.com

  deploy-production:
    needs: build
    runs-on: ubuntu-latest
    if: startsWith(github.ref, 'refs/tags/v')
    environment:
      name: production
      url: https://api.example.com

    steps:
      - uses: actions/checkout@v3

      - name: Set up kubectl
        uses: azure/setup-kubectl@v3

      - name: Configure kubectl
        run: |
          echo "${{ secrets.KUBE_CONFIG_PROD }}" | base64 -d > kubeconfig
          export KUBECONFIG=kubeconfig

      - name: Deploy to production
        run: |
          kubectl set image deployment/agent-service \
            agent-service=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${GITHUB_REF##*/} \
            -n production
          kubectl rollout status deployment/agent-service -n production

      - name: Verify deployment
        run: |
          ./scripts/verify-deployment.sh https://api.example.com
```

## 📦 灰度发布

### Canary Deployment

```yaml
# canary-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-service-canary
  namespace: production
spec:
  replicas: 1 # 金丝雀副本数
  selector:
    matchLabels:
      app: agent-service
      track: canary
  template:
    metadata:
      labels:
        app: agent-service
        track: canary
        version: v1.1.0
    spec:
      containers:
        - name: agent-service
          image: your-registry/agent-service:v1.1.0
          # ... 其他配置同主 Deployment
```

### Ingress 流量分配

```yaml
# ingress-canary.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: agent-service-canary
  namespace: production
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "10" # 10% 流量到金丝雀
spec:
  ingressClassName: nginx
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: agent-service-canary
                port:
                  name: http
```

### 灰度发布流程

```bash
#!/bin/bash
# canary-rollout.sh

# 1. 部署金丝雀版本 (10% 流量)
kubectl apply -f canary-deployment.yaml
kubectl set image deployment/agent-service-canary \
  agent-service=your-registry/agent-service:v1.1.0 \
  -n production

# 等待金丝雀就绪
kubectl rollout status deployment/agent-service-canary -n production

# 配置 10% 流量到金丝雀
kubectl apply -f ingress-canary.yaml

# 2. 监控金丝雀指标 (15分钟)
echo "Monitoring canary for 15 minutes..."
sleep 900

# 检查错误率
ERROR_RATE=$(curl -s 'http://prometheus:9090/api/v1/query?query=rate(agent_errors_total{track="canary"}[5m])' | jq -r '.data.result[0].value[1]')

if (( $(echo "$ERROR_RATE > 0.01" | bc -l) )); then
  echo "Canary error rate too high: $ERROR_RATE"
  echo "Rolling back..."
  kubectl delete deployment agent-service-canary -n production
  kubectl delete ingress agent-service-canary -n production
  exit 1
fi

# 3. 逐步增加流量
for weight in 25 50 75 100; do
  echo "Increasing canary traffic to $weight%..."
  kubectl annotate ingress agent-service-canary \
    nginx.ingress.kubernetes.io/canary-weight=$weight \
    --overwrite -n production

  echo "Monitoring for 10 minutes..."
  sleep 600

  # 检查指标
  # ...
done

# 4. 全量发布
echo "Canary successful, promoting to stable..."
kubectl set image deployment/agent-service \
  agent-service=your-registry/agent-service:v1.1.0 \
  -n production

kubectl rollout status deployment/agent-service -n production

# 清理金丝雀
kubectl delete deployment agent-service-canary -n production
kubectl delete ingress agent-service-canary -n production

echo "Deployment complete!"
```

## 🔙 回滚策略

### 快速回滚

```bash
# 查看部署历史
kubectl rollout history deployment/agent-service -n production

# 回滚到上一个版本
kubectl rollout undo deployment/agent-service -n production

# 回滚到指定版本
kubectl rollout undo deployment/agent-service \
  --to-revision=3 -n production

# 暂停部署
kubectl rollout pause deployment/agent-service -n production

# 恢复部署
kubectl rollout resume deployment/agent-service -n production
```

### 自动回滚脚本

```bash
#!/bin/bash
# auto-rollback.sh

DEPLOYMENT="agent-service"
NAMESPACE="production"
ERROR_THRESHOLD=0.05
DURATION=300  # 5分钟

echo "Monitoring deployment for $DURATION seconds..."

start_time=$(date +%s)
while true; do
  current_time=$(date +%s)
  elapsed=$((current_time - start_time))

  if [ $elapsed -gt $DURATION ]; then
    echo "Monitoring period complete. Deployment successful!"
    exit 0
  fi

  # 检查错误率
  error_rate=$(curl -s 'http://prometheus:9090/api/v1/query?query=rate(agent_errors_total[1m])' | \
    jq -r '.data.result[0].value[1] // 0')

  echo "Current error rate: $error_rate"

  if (( $(echo "$error_rate > $ERROR_THRESHOLD" | bc -l) )); then
    echo "ERROR: Error rate $error_rate exceeds threshold $ERROR_THRESHOLD"
    echo "Initiating automatic rollback..."

    kubectl rollout undo deployment/$DEPLOYMENT -n $NAMESPACE
    kubectl rollout status deployment/$DEPLOYMENT -n $NAMESPACE

    echo "Rollback complete!"
    exit 1
  fi

  sleep 10
done
```

## 📋 部署检查清单

### 上线前检查

- [ ] 代码审查通过
- [ ] 所有测试通过(单元、集成、E2E)
- [ ] 测试覆盖率 >= 80%
- [ ] 安全扫描通过
- [ ] 性能测试通过
- [ ] 文档已更新
- [ ] 变更日志已记录
- [ ] 数据库迁移脚本已准备
- [ ] 配置文件已审核
- [ ] 回滚方案已准备

### 部署中检查

- [ ] 健康检查通过
- [ ] 就绪检查通过
- [ ] 监控指标正常
- [ ] 日志无异常
- [ ] 错误率在阈值内
- [ ] 响应时间正常
- [ ] 资源使用正常

### 部署后验证

- [ ] 冒烟测试通过
- [ ] 端到端测试通过
- [ ] 监控告警配置正确
- [ ] 日志收集正常
- [ ] 追踪系统正常
- [ ] 备份策略生效
- [ ] 文档已同步

## 🔗 相关资源

- [监控运维](/best-practices/monitoring)
- [性能优化](/best-practices/performance)
- [安全建议](/best-practices/security)
- [Kubernetes 文档](https://kubernetes.io/docs/)
- [Docker 最佳实践](https://docs.docker.com/develop/dev-best-practices/)

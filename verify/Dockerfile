# verify：一次性验算服务。
# 依次跑齐 api 的 go vet + 单元测试 + 生产构建，
# 以及 web 的单元测试 + 类型检查 + 生产构建；任一环节失败则构建失败。

FROM golang:1.22-alpine AS api
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ .
RUN go vet ./... \
    && go test ./... \
    && CGO_ENABLED=0 go build -o /out/api .

FROM node:20-alpine AS web
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run test && npm run build

FROM alpine:3.20
COPY --from=api /out/api /artifacts/api
COPY --from=web /app/dist /artifacts/web-dist
CMD ["sh", "-c", "echo 'verify OK：api 测试与构建通过；web 测试与生产构建通过（产物在 /artifacts）'"]

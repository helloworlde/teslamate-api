# TeslaMateApi

[![GitHub CI](https://img.shields.io/github/actions/workflow/status/tobiasehlert/teslamateapi/build.yml?branch=main&logo=github)](https://github.com/tobiasehlert/teslamateapi/actions/workflows/build.yml)
[![GitHub go.mod version](https://img.shields.io/github/go-mod/go-version/tobiasehlert/teslamateapi?logo=go)](https://github.com/tobiasehlert/teslamateapi/blob/main/go.mod)
[![GitHub release](https://img.shields.io/github/v/release/tobiasehlert/teslamateapi?sort=semver&logo=github)](https://github.com/tobiasehlert/teslamateapi/releases)
[![Docker image size (tag)](https://img.shields.io/docker/image-size/tobiasehlert/teslamateapi/latest?logo=docker)](https://hub.docker.com/r/tobiasehlert/teslamateapi)
[![GitHub license](https://img.shields.io/github/license/tobiasehlert/teslamateapi)](https://github.com/tobiasehlert/teslamateapi/blob/main/LICENSE)
[![Docker pulls](https://img.shields.io/docker/pulls/tobiasehlert/teslamateapi)](https://hub.docker.com/r/tobiasehlert/teslamateapi)

TeslaMateApi is a RESTful API to get data collected by self-hosted data logger **[TeslaMate](https://github.com/teslamate-org/teslamate)** in JSON.

- Written in **[Golang](https://golang.org/)**
- Data is collected from TeslaMate **Postgres** database and local **MQTT** Broker
- Endpoints return data in JSON format
- **Read-only**：不向 Tesla 账户转发遥控指令，也不代理 TeslaMate 内部日志接口

### Table of Contents

- [How to use](#how-to-use)
  - [Docker-compose](#docker-compose)
  - [Environment variables](#environment-variables)
- [API documentation](#api-documentation)
  - [Available endpoints](#available-endpoints)
- [Security information](#security-information)
- [Credits](#credits)

## How to use

You can either use it in a Docker container or go download the code and deploy it yourself on any server.

### Docker-compose

If you run the simple Docker deployment of TeslaMate, then adding this will do the trick. You'll have TeslaMateApi exposed at port 8080 locally then.

```yaml
services:
  teslamateapi:
    image: tobiasehlert/teslamateapi:latest
    restart: always
    depends_on:
      - database
    environment:
      - DATABASE_USER=teslamate
      - DATABASE_PASS=secret
      - DATABASE_NAME=teslamate
      - DATABASE_HOST=database
      - MQTT_HOST=mosquitto
      - TZ=Europe/Berlin
    ports:
      - 8080:8080
```

If you are using TeslaMate Traefik setup in Docker with environment variables file (.env), then you can simply add this section to the `services:` section of the `docker-compose.yml` file:

```yaml
services:
  teslamateapi:
    image: tobiasehlert/teslamateapi:latest
    restart: always
    depends_on:
      - database
    environment:
      - DATABASE_USER=${TM_DB_USER}
      - DATABASE_PASS=${TM_DB_PASS}
      - DATABASE_NAME=${TM_DB_NAME}
      - DATABASE_HOST=database
      - MQTT_HOST=mosquitto
      - TZ=${TM_TZ}
    labels:
      - "traefik.enable=true"
      - "traefik.port=8080"
      - "traefik.http.middlewares.redirect.redirectscheme.scheme=https"
      - "traefik.http.middlewares.teslamateapi-auth.basicauth.realm=teslamateapi"
      - "traefik.http.middlewares.teslamateapi-auth.basicauth.usersfile=/auth/.htpasswd"
      - "traefik.http.routers.teslamateapi-insecure.rule=Host(`${FQDN_TM}`)"
      - "traefik.http.routers.teslamateapi-insecure.middlewares=redirect"
      - "traefik.http.routers.teslamateapi.rule=Host(`${FQDN_TM}`) && (Path(`/api`) || PathPrefix(`/api/`))"
      - "traefik.http.routers.teslamateapi.entrypoints=websecure"
      - "traefik.http.routers.teslamateapi.middlewares=teslamateapi-auth"
      - "traefik.http.routers.teslamateapi.tls.certresolver=tmhttpchallenge"
```

In this case, the TeslaMateApi would be accessible at teslamate.example.com/api/

### Environment variables

Basically the same environment variables for the database, mqqt and timezone need to be set for TeslaMateApi as you have for TeslaMate.

**Required** environment variables (even if there are some default values available)

| Variable           | Type   | Default         |
| ------------------ | ------ | --------------- |
| **DATABASE_USER**  | string | _teslamate_     |
| **DATABASE_PASS**  | string | _secret_        |
| **DATABASE_NAME**  | string | _teslamate_     |
| **DATABASE_HOST**  | string | _database_      |
| **MQTT_HOST**      | string | _mosquitto_     |
| **TZ**             | string | _Europe/Berlin_ |

**Optional** environment variables

| Variable                      | Type    | Default                       |
| ----------------------------- | ------- | ----------------------------- |
| **DATABASE_PORT**             | integer | _5432_                        |
| **DATABASE_TIMEOUT**          | integer | _60000_                       |
| **DATABASE_SSL**              | string  | _disable_                     |
| **DATABASE_SSL_CA_CERT_FILE** | string  |                               |
| **DEBUG_MODE**                | boolean | _false_                       |
| **DISABLE_MQTT**              | boolean | _false_                       |
| **MQTT_TLS**                  | boolean | _false_                       |
| **MQTT_PORT**                 | integer | _1883 (if TLS is true: 8883)_ |
| **MQTT_USERNAME**             | string  |                               |
| **MQTT_PASSWORD**             | string  |                               |
| **MQTT_NAMESPACE**            | string  |                               |
| **MQTT_CLIENTID**             | string  | _4 char random string_        |
| **TESLA_API_HOST**            | string  | _若设置则启动时写一条 info 日志；无其它逻辑_   |

## API documentation

交互式 OpenAPI（Swagger UI）在运行服务后访问：**`http://localhost:8080/swagger`** 或 **`/swagger/index.html`**（若端口或主机不同请相应替换；仅打开根路径 `/` 只会看到 JSON 提示，不是文档页）。规范文件由 [swag](https://github.com/swaggo/swag) 根据源码注释生成，位于 `src/docs/swagger.json`。修改接口后请在仓库根目录执行：

```bash
swag init -g webserver.go -d src -o src/docs --parseDependency --parseInternal
```

更细的分阶段看板与 API 对照见仓库根目录 **`执行计划.md`**。各看板文件、面板与接口的逐项说明见 **`dashboards/CATALOG.md`**。**按功能聚合的去重视图**（含 `charging-stats` / `extra` 等同域关系、已废弃路径说明）见 **`dashboards/API_AGGREGATE.md`**。

### Available endpoints

前缀 **`/api/v1`**（另有 **`/api`**、**`/api/ping`**、**`/api/healthz`**、**`/api/readyz`**）。

| 分组 | 路径模式 |
|------|----------|
| 服务信息 | `GET /api`、`GET /api/v1` |
| 全局 | `GET /database`、`GET /globalsettings`、`GET /cars` |
| 单车 | `GET /cars/:CarID`、`battery-health`、`states`、`positions`、`status`、`updates` |
| 充电 / 行程 | `GET .../charges`（`startDate`/`endDate`）、`.../charges/current`、`.../charges/:ChargeID`；`GET .../drives`（同上 + `minDistance`/`maxDistance`）、`.../drives/:DriveID` |
| 看板 metrics | `GET .../metrics/<name>`，含 `charging-stats` + `charging-stats/extra`、`drive-stats` + `drive-stats/extra`，以及 `efficiency`、`mileage`、`locations`、`timeline`、`vampire-drain`、`statistics`、`charge-level`、`projected-range`、`overview`、`states-analytics`、`visited`、`dutch-tax`、`trip` |

完整参数与各端点说明以 **Swagger UI**（`/swagger/index.html`）及 **`src/docs/swagger.json`** 为准。

> [!TIP]
> Canonical UTC format in RFC3339, e.g. `2006-01-02T15:04:05Z` or `2006-01-02T15:04:05+07:00`

## Security information

本服务为**只读**数据 API：通过 Postgres 与 MQTT 暴露 TeslaMate 已采集的数据，**不**实现车辆遥控、**不**解密或转发 Tesla 账户令牌、**不**调用 Tesla Fleet 指令接口。

请在反向代理或网关层对 `/api` 做访问控制（认证、TLS、限流等）。上文 Traefik 示例使用与 TeslaMate 相同的 Basic Auth，仅为一种常见做法。

## Credits

- Authors: Tobias Lindberg – [List of contributors](https://github.com/tobiasehlert/teslamateapi/graphs/contributors)
- Distributed under MIT License

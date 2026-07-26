# GoBook HTTP API 接口文档

> 依据当前工作区源码生成，基准提交：`023894e`，生成日期：2026-07-23。  
> 本文记录的是代码当前实际行为，包括尚未统一的响应格式和已知风险，不代表理想化设计。

## 1. 服务概览

| 项目 | 地址 | 说明 |
| --- | --- | --- |
| 业务 API | `http://localhost:8080` | Gin HTTP 服务 |
| Prometheus | `http://127.0.0.1:8081/metrics` | 仅绑定本机地址 |
| 默认请求类型 | `application/json` | POST 请求体主要通过 Gin `Bind` 解析 |
| 认证方式 | `Authorization: Bearer <token>` | JWT Bearer Token |

当前对外注册了 20 个 Gin 路由，另有 1 个独立的 Prometheus 指标入口。

## 2. 接口总览

### 2.1 用户接口

| 方法 | 路径 | 是否需要登录 | 响应格式 | 说明 |
| --- | --- | --- | --- | --- |
| POST | `/users/signup` | 否 | 文本 | 邮箱注册 |
| POST | `/users/login` | 否 | `Result` | 邮箱密码登录 |
| POST | `/users/sendSMSCode` | 否 | `Result` | 发送短信验证码 |
| POST | `/users/loginSMS` | 否 | `Result` | 短信验证码登录或注册 |
| POST | `/users/refreshToken` | 使用刷新令牌 | `Result` | 刷新访问令牌 |
| POST | `/users/logout` | 是 | `Result` | 注销当前 JWT 会话 |
| GET | `/users/profile` | 是 | 自定义 JSON | 获取当前用户资料 |
| POST | `/users/create` | 是 | 文本 | 占位接口 |
| POST | `/users/delete` | 是 | 文本 | 占位接口 |
| POST | `/users/edit` | 是 | 文本 | 占位接口 |

### 2.2 文章接口

| 方法 | 路径 | 是否需要登录 | 响应格式 | 说明 |
| --- | --- | --- | --- | --- |
| POST | `/articles/edit` | 是 | `Result` | 新建或编辑文章草稿 |
| POST | `/articles/publish` | 是 | `Result` | 发表文章 |
| POST | `/articles/withdraw` | 是 | `Result` | 撤回已发表文章 |
| POST | `/articles/list` | 是 | `Result` | 查询当前作者的文章列表 |
| GET | `/articles/detail/:id` | 是 | `Result` | 获取制作库中的文章详情 |
| GET | `/pub/:id` | 是 | `Result` | 获取线上库中的已发表文章 |
| POST | `/pub/like` | 是 | `Result` | 点赞文章 |

### 2.3 微信 OAuth2

| 方法 | 路径 | 是否需要登录 | 响应格式 | 说明 |
| --- | --- | --- | --- | --- |
| GET | `/oauth2/wechat/authurl` | 否 | `Result` | 获取微信扫码登录地址 |
| GET | `/oauth2/wechat/callback` | 否 | `Result` | 微信扫码登录回调 |

### 2.4 可观测性

| 方法 | 地址 | 是否需要登录 | 响应格式 | 说明 |
| --- | --- | --- | --- | --- |
| GET | `http://localhost:8080/test/metrics` | 否 | 文本 | 产生随机延迟以测试指标 |
| GET | `http://127.0.0.1:8081/metrics` | 否 | Prometheus 文本 | 指标采集入口 |

## 3. 公共约定

### 3.1 统一业务响应

大部分接口使用以下 JSON 结构：

```json
{
  "code": 0,
  "msg": "操作成功",
  "data": null
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | integer | 业务状态码 |
| `msg` | string | 用户提示或错误信息 |
| `data` | any | 业务数据；无数据时通常为 `null` |

当前代码中使用的主要业务码：

| 业务码 | 含义 |
| --- | --- |
| `0` | 成功 |
| `4` | 参数、验证码或 OAuth state 校验错误 |
| `5` | 业务失败或系统错误 |

注意：业务失败通常仍返回 HTTP `200`，调用方需要同时检查 HTTP 状态码和业务 `code`。

### 3.2 HTTP 状态码

| HTTP 状态码 | 当前使用场景 |
| --- | --- |
| `200 OK` | 成功，以及大部分业务错误 |
| `400 Bad Request` | Gin 请求体绑定失败时可能自动返回 |
| `401 Unauthorized` | JWT 缺失、无效、过期、User-Agent 不一致或会话已注销 |
| `429 Too Many Requests` | IP 触发限流 |
| `500 Internal Server Error` | 限流 Redis 异常、部分详情接口内部错误 |

### 3.3 JWT 认证

登录成功后，服务通过响应头返回两种令牌：

| 响应头 | 有效期 | 用途 |
| --- | --- | --- |
| `x-jwt-token` | 60 分钟 | 访问受保护接口 |
| `x-refresh-token` | 7 天 | 获取新的访问令牌 |

调用受保护接口：

```http
Authorization: Bearer <x-jwt-token>
```

刷新访问令牌：

```http
Authorization: Bearer <x-refresh-token>
```

认证中间件还会进行以下检查：

1. JWT 签名与有效期有效。
2. JWT 中的用户 ID 不为 `0`。
3. 请求 `User-Agent` 必须与登录时完全一致。
4. JWT 中的会话 ID（SSID）不能存在于 Redis 注销黑名单。

因此，客户端刷新或复用令牌时应保持相同的 `User-Agent`。

### 3.4 无需访问令牌的路径

以下路径被 JWT 中间件明确忽略：

```text
/users/login
/users/signup
/users/sendSMSCode
/users/loginSMS
/users/refreshToken
/oauth2/wechat/authurl
/oauth2/wechat/callback
/test/metrics
```

除此之外的 Gin 路由当前都需要有效访问令牌，包括 `GET /pub/:id`。

### 3.5 限流

所有通过 Gin 的请求按客户端 IP 限流：

- 时间窗口：1 秒
- 最大请求数：100
- 超限响应：HTTP `429`
- 限流依赖 Redis；Redis 异常时返回 HTTP `500`

### 3.6 CORS

- 允许方法：`GET`、`POST`、`PUT`、`DELETE`、`PATCH`、`OPTIONS`
- 允许请求头：`Origin`、`Content-Type`、`Accept`、`Authorization`、`X-Requested-With`
- 暴露响应头：`X-Total-Count`、`X-JWT-Token`、`x-refresh-token`
- 允许携带 Cookie
- 允许以 `http://localhost` 开头的 Origin
- 允许包含 `xxx.com` 的 Origin
- 预检缓存时间：12 小时

## 4. 数据模型

### 4.1 ArticleReq

用于编辑和发表文章。

```json
{
  "id": 0,
  "title": "Go 并发编程",
  "content": "文章正文"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | int64 | 否 | `0` 表示新建；大于 `0` 表示更新已有文章 |
| `title` | string | 是 | 标题；Handler 当前未做非空校验 |
| `content` | string | 是 | 正文；Handler 当前未做非空校验 |

### 4.2 ArticleVO

文章响应对象会根据接口场景返回部分字段：

```json
{
  "id": 1001,
  "title": "Go 并发编程",
  "abstract": "正文前一百个字符",
  "content": "完整正文",
  "authorId": 12,
  "authorName": "作者",
  "status": 2,
  "ctime": "2026-07-23 10:30:00",
  "utime": "2026-07-23 11:00:00",
  "readCnt": 100,
  "likeCnt": 20,
  "collectCnt": 5,
  "liked": false,
  "collected": false
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | int64 | 文章 ID |
| `title` | string | 标题 |
| `abstract` | string | 按 Unicode 字符截取的正文前 100 字 |
| `content` | string | 完整正文 |
| `authorId` | int64 | 作者 ID |
| `authorName` | string | 作者名称 |
| `status` | uint8 | `0` 未知、`1` 未发表、`2` 已发表、`3` 私密 |
| `ctime` | string | 创建时间，格式 `YYYY-MM-DD HH:mm:ss` |
| `utime` | string | 更新时间，格式 `YYYY-MM-DD HH:mm:ss` |
| `readCnt` | int64 | 阅读数 |
| `likeCnt` | int64 | 点赞数 |
| `collectCnt` | int64 | 收藏数 |
| `liked` | boolean | 当前用户是否点赞；当前详情组装代码未赋值，通常为 `false` |
| `collected` | boolean | 当前用户是否收藏；当前详情组装代码未赋值，通常为 `false` |

多数基础字段带有 `omitempty`，零值字段可能不出现在 JSON 中；互动数字和布尔字段没有 `omitempty`。

## 5. 用户接口

### 5.1 邮箱注册

```http
POST /users/signup
Content-Type: application/json
```

认证：不需要。

请求体：

```json
{
  "email": "user@example.com",
  "password": "Passw0rd!",
  "confirm_password": "Passw0rd!"
}
```

| 字段 | 类型 | 必填 | 校验 |
| --- | --- | --- | --- |
| `email` | string | 是 | 必须匹配邮箱正则 |
| `password` | string | 是 | 至少 8 位，包含字母、数字及特殊字符 `$ @ ! % * # ? &` |
| `confirm_password` | string | 是 | 必须与 `password` 相同 |

成功响应为纯文本：

```text
SignUp success~
```

可能的 HTTP `200` 文本响应：

```text
邮箱格式错误！
两次输入的密码不一致
密码必须大于8位，包含数字、特殊字符
邮箱重复，请换一个！
系统错误
```

示例：

```bash
curl -X POST 'http://localhost:8080/users/signup' \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "user@example.com",
    "password": "Passw0rd!",
    "confirm_password": "Passw0rd!"
  }'
```

### 5.2 邮箱密码登录

```http
POST /users/login
Content-Type: application/json
```

认证：不需要。

请求体：

```json
{
  "email": "user@example.com",
  "password": "Passw0rd!"
}
```

成功响应头：

```http
x-jwt-token: <access-token>
x-refresh-token: <refresh-token>
```

成功响应：

```json
{
  "code": 0,
  "msg": "登录成功～",
  "data": null
}
```

失败响应示例：

```json
{
  "code": 5,
  "msg": "账号/邮箱或密码错误！",
  "data": null
}
```

还可能返回 `"用户不存在！"` 或 `"系统错误！"`。

示例：

```bash
curl -i -X POST 'http://localhost:8080/users/login' \
  -H 'Content-Type: application/json' \
  -H 'User-Agent: GoBook-Web/1.0' \
  -d '{"email":"user@example.com","password":"Passw0rd!"}'
```

### 5.3 发送短信验证码

```http
POST /users/sendSMSCode
Content-Type: application/json
```

认证：不需要。

请求体：

```json
{
  "phone": "13800138000"
}
```

成功响应：

```json
{
  "code": 0,
  "msg": "发送成功～",
  "data": null
}
```

失败消息：

- `短信发送频繁，请稍后再试！`
- `系统异常!`

当前 Handler 未进行手机号格式校验。

### 5.4 短信验证码登录

```http
POST /users/loginSMS
Content-Type: application/json
```

认证：不需要。

请求体：

```json
{
  "phone": "13800138000",
  "code": "123456"
}
```

处理流程：

1. 以业务标识 `login` 校验短信验证码。
2. 根据手机号查询用户。
3. 用户不存在时自动创建。
4. 生成访问令牌和刷新令牌。

成功响应头与邮箱登录相同。

成功响应：

```json
{
  "code": 0,
  "msg": "登录成功～",
  "data": null
}
```

失败业务码：

| code | msg 示例 |
| --- | --- |
| `4` | `验证码错误!` |
| `5` | `验证码校验错误，请重新获取验证码` |
| `5` | `手机号码登录失败` |
| `5` | `系统错误！` |

### 5.5 刷新访问令牌

```http
POST /users/refreshToken
Authorization: Bearer <refresh-token>
```

该路径不经过访问令牌中间件，但必须在 `Authorization` 中传入有效刷新令牌。

成功响应头：

```http
x-jwt-token: <new-access-token>
```

成功响应：

```json
{
  "code": 0,
  "msg": "token刷新成功～",
  "data": null
}
```

以下情况返回 HTTP `401`，响应体可能为空：

- 刷新令牌缺失、格式错误、过期或签名错误
- 对应 SSID 已注销
- Redis 会话检查失败
- 新访问令牌生成失败

示例：

```bash
curl -i -X POST 'http://localhost:8080/users/refreshToken' \
  -H 'Authorization: Bearer <refresh-token>' \
  -H 'User-Agent: GoBook-Web/1.0'
```

### 5.6 注销当前会话

```http
POST /users/logout
Authorization: Bearer <access-token>
```

处理行为：

1. 将当前 SSID 写入 Redis 注销黑名单，保留 7 天。
2. 在响应头中把 `X-JWT-Token` 和 `x-refresh-token` 置空。
3. 后续使用同一 SSID 的访问令牌或刷新令牌会被拒绝。

成功响应：

```json
{
  "code": 0,
  "msg": "退出登录成功～",
  "data": null
}
```

失败响应：

```json
{
  "code": 5,
  "msg": "退出登录失败！",
  "data": null
}
```

### 5.7 获取当前用户资料

```http
GET /users/profile
Authorization: Bearer <access-token>
```

成功响应结构与统一 `Result` 不同：

```json
{
  "success": true,
  "data": {
    "Id": 12,
    "Email": "user@example.com",
    "Password": "$2a$10$...",
    "Phone": "13800138000",
    "WechatInfo": {
      "openid": "",
      "unionid": ""
    },
    "Ctime": "2026-07-23T10:00:00+08:00"
  },
  "code": 200
}
```

当前 `domain.User` 没有 JSON 标签，字段名保持 Go 导出字段形式。代码还会返回 `Password` 字段，其中通常是 bcrypt 哈希；该字段不应暴露给客户端，参见“已知问题”。

失败时可能返回：

- HTTP `401`：认证失败
- HTTP `500`，文本 `获取用户信息失败！`
- HTTP `200`，文本 `系统错误`：上下文 claims 异常

### 5.8 用户占位接口

以下接口均需要访问令牌，但当前没有实际 CRUD 逻辑，也不解析请求体。

| 方法 | 路径 | HTTP 200 响应 |
| --- | --- | --- |
| POST | `/users/create` | `CreateSuccess~` |
| POST | `/users/delete` | `DeleteSuccess~` |
| POST | `/users/edit` | `EditSuccess~` |

## 6. 文章接口

所有文章接口当前都需要：

```http
Authorization: Bearer <access-token>
```

### 6.1 新建或编辑草稿

```http
POST /articles/edit
Content-Type: application/json
```

请求体：

```json
{
  "id": 0,
  "title": "Go 并发编程",
  "content": "文章正文"
}
```

- `id = 0`：创建新草稿。
- `id > 0`：更新已有文章。
- 作者 ID 从 JWT 中取得，客户端无需传入。
- Service 强制把状态设为 `1`（未发表）。

成功响应：

```json
{
  "code": 0,
  "msg": "编辑成功～",
  "data": 1001
}
```

### 6.2 发表文章

```http
POST /articles/publish
Content-Type: application/json
```

请求体与 `/articles/edit` 相同。

处理行为：

- Service 强制把文章状态设为 `2`（已发表）。
- 同步写入制作库和线上库。

成功响应：

```json
{
  "code": 0,
  "msg": "发表成功～",
  "data": 1001
}
```

### 6.3 撤回文章

```http
POST /articles/withdraw
Content-Type: application/json
```

请求体：

```json
{
  "id": 1001
}
```

Service 使用 JWT 用户 ID 校验归属，并将制作库和线上库中的状态同步为未发表。

成功响应：

```json
{
  "code": 0,
  "msg": "文章撤回成功～",
  "data": 1001
}
```

### 6.4 查询当前作者的文章列表

```http
POST /articles/list
Content-Type: application/json
```

请求体：

```json
{
  "offset": 0,
  "limit": 20
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `offset` | integer | 跳过的记录数 |
| `limit` | integer | 返回数量 |

当前 Handler 未限制负数、零值或最大 `limit`。

成功响应：

```json
{
  "code": 0,
  "msg": "",
  "data": [
    {
      "id": 1001,
      "title": "Go 并发编程",
      "abstract": "文章正文摘要",
      "status": 1,
      "ctime": "2026-07-23 10:30:00",
      "utime": "2026-07-23 11:00:00",
      "readCnt": 0,
      "likeCnt": 0,
      "collectCnt": 0,
      "liked": false,
      "collected": false
    }
  ]
}
```

列表不返回完整 `content`，摘要最多为正文前 100 个 Unicode 字符。

### 6.5 获取作者视角的文章详情

```http
GET /articles/detail/:id
```

路径参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | int64 | 文章 ID |

数据从制作库读取，可包含未发表草稿。

成功响应：

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "id": 1001,
    "title": "Go 并发编程",
    "content": "完整正文",
    "status": 1,
    "ctime": "2026-07-23 10:30:00",
    "utime": "2026-07-23 11:00:00",
    "readCnt": 0,
    "likeCnt": 0,
    "collectCnt": 0,
    "liked": false,
    "collected": false
  }
}
```

当前实现中，无法解析 `id` 时返回 HTTP `500` 和业务码 `4`，而不是 HTTP `400`。

### 6.6 获取已发表文章详情

```http
GET /pub/:id
```

注意：尽管该路径位于 `/pub` 分组，当前仍需要访问令牌。

处理流程：

1. 从线上库查询已发表文章。
2. 异步发送 Kafka 阅读事件。
3. 查询阅读数、点赞数和收藏数。
4. 组装文章详情并立即返回。

阅读数采用最终一致模型，本次响应中的 `readCnt` 可能尚未包含当前访问。

成功响应：

```json
{
  "code": 0,
  "msg": "OK~",
  "data": {
    "id": 1001,
    "title": "Go 并发编程",
    "content": "完整正文",
    "status": 2,
    "ctime": "2026-07-23 10:30:00",
    "utime": "2026-07-23 11:00:00",
    "readCnt": 101,
    "likeCnt": 20,
    "collectCnt": 5,
    "liked": false,
    "collected": false
  }
}
```

错误行为：

- `id` 无法解析：HTTP `200`，业务码 `4`
- 文章或互动数据查询失败：HTTP `500`，业务码 `5`

### 6.7 点赞文章

```http
POST /pub/like
Content-Type: application/json
```

请求体：

```json
{
  "id": 1001
}
```

用户 ID 从 JWT 获取。当前 Service 调用的是 `IncrLike`：

- 重复点赞依靠数据库唯一索引实现幂等，不会重复累计。
- 当前没有注册取消点赞的 HTTP 路由。
- Handler 注释中提到“点赞/取消点赞切换”，但实际代码只执行点赞。

成功响应：

```json
{
  "code": 0,
  "msg": "点赞成功～",
  "data": null
}
```

## 7. 微信 OAuth2 接口

### 7.1 获取微信授权地址

```http
GET /oauth2/wechat/authurl
```

认证：不需要。

成功时，服务会：

1. 生成随机 `state`。
2. 将带签名的 state JWT 写入 `jwt-state` Cookie。
3. 返回微信扫码授权地址。

Cookie 属性：

| 属性 | 值 |
| --- | --- |
| 名称 | `jwt-state` |
| 有效期 | 600 秒 |
| Path | `/oauth2/wechat/callback` |
| HttpOnly | `true` |
| Secure | 由 `WechatHandlerConfig.Secure` 决定 |

成功响应：

```json
{
  "code": 0,
  "msg": "",
  "data": "https://open.weixin.qq.com/connect/qrconnect?..."
}
```

示例：

```bash
curl -i -c cookies.txt \
  'http://localhost:8080/oauth2/wechat/authurl'
```

### 7.2 微信登录回调

```http
GET /oauth2/wechat/callback?code=<code>&state=<state>
```

认证：不需要访问令牌，但需要浏览器携带上一步设置的 `jwt-state` Cookie。

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `code` | string | 是 | 微信返回的一次性授权码 |
| `state` | string | 是 | 必须与 Cookie 中保存的 state 一致 |

处理流程：

1. 验证 `jwt-state` Cookie 的签名和有效期。
2. 比较查询参数 `state` 与 Cookie 中的 state。
3. 使用 `code` 向微信换取 OpenID/UnionID。
4. 查找或创建 GoBook 用户。
5. 在响应头中返回访问令牌和刷新令牌。

成功响应头：

```http
x-jwt-token: <access-token>
x-refresh-token: <refresh-token>
```

成功响应：

```json
{
  "code": 0,
  "msg": "OK~",
  "data": null
}
```

state 校验失败时返回业务码 `4`；微信请求、用户创建或令牌生成失败时返回业务码 `5`。

示例：

```bash
curl -i -b cookies.txt \
  'http://localhost:8080/oauth2/wechat/callback?code=WECHAT_CODE&state=STATE'
```

## 8. 可观测性接口

### 8.1 指标延迟测试

```http
GET /test/metrics
```

认证：不需要。

接口随机等待 `0–999ms`，用于产生不同响应耗时，验证 Prometheus HTTP 指标。

响应：

```text
OK~
```

### 8.2 Prometheus 指标

```http
GET http://127.0.0.1:8081/metrics
```

该服务由独立的 `net/http` Server 提供，只监听 `127.0.0.1:8081`，不经过 Gin 的 JWT、CORS 或限流中间件。

Gin 请求耗时指标按以下标签区分：

- `pattern`：路由模板，例如 `/pub/:id`
- `method`：HTTP 方法
- `status`：HTTP 状态码
- `instance_id`：固定实例标识

## 9. 通用调用示例

### 9.1 登录并提取令牌

```bash
curl -i -X POST 'http://localhost:8080/users/login' \
  -H 'Content-Type: application/json' \
  -H 'User-Agent: GoBook-Web/1.0' \
  -d '{"email":"user@example.com","password":"Passw0rd!"}'
```

从响应头保存：

```text
x-jwt-token
x-refresh-token
```

### 9.2 携带访问令牌查询文章

```bash
curl 'http://localhost:8080/articles/list' \
  -X POST \
  -H 'Authorization: Bearer <access-token>' \
  -H 'User-Agent: GoBook-Web/1.0' \
  -H 'Content-Type: application/json' \
  -d '{"offset":0,"limit":20}'
```

### 9.3 使用刷新令牌

```bash
curl -i 'http://localhost:8080/users/refreshToken' \
  -X POST \
  -H 'Authorization: Bearer <refresh-token>' \
  -H 'User-Agent: GoBook-Web/1.0'
```

## 10. 当前实现的已知问题

以下内容来自源码扫描，建议在对外发布 API 前处理：

1. **用户资料可能泄露密码哈希**  
   `/users/profile` 直接序列化 `domain.User`，其中包含 `Password`。应使用专用 UserVO，并彻底排除密码字段。

2. **公开文章详情实际需要登录**  
   `/pub/:id` 没有加入 JWT 忽略列表。如果希望游客阅读，应明确加入白名单，并调整阅读事件中匿名用户的处理方式。

3. **作者详情缺少显式归属参数**  
   `/articles/detail/:id` 的 Handler 只按文章 ID 查询，没有把 JWT 用户 ID 传入 Service。需要确认 Repository 是否能阻止越权读取其他作者的草稿。

4. **业务错误与 HTTP 状态码混用**  
   大量业务失败返回 HTTP `200`；部分参数错误返回 HTTP `500`。建议统一 HTTP 状态和业务码语义。

5. **注册接口响应格式不统一**  
   `/users/signup` 返回纯文本，而登录、短信和文章接口返回 `Result` JSON。

6. **占位接口已注册**  
   `/users/create`、`/users/delete`、`/users/edit` 会返回成功文本，但没有实际业务操作，容易让调用方误判。

7. **点赞接口不支持取消**  
   当前 `/pub/like` 仅执行幂等点赞，没有取消点赞路由，与 Handler 中“切换”的注释不一致。

8. **互动状态尚未组装**  
   已发表文章详情的 `liked`、`collected` 字段当前没有赋值，通常始终为 `false`。

9. **分页参数未校验**  
   `/articles/list` 未限制负数、零值和最大 `limit`。

10. **短信登录错误分支可能重复写响应**  
    验证次数耗尽分支直接调用 `ctx.JSON` 后没有立即返回，后续逻辑仍可能继续执行。

11. **OAuth 错误信息可能暴露内部细节**  
    state 校验失败响应会包含底层错误或实际 state 内容，建议对客户端返回固定消息，详细信息只写服务端日志。

12. **微信响应体未显式关闭**  
    微信 OAuth Service 请求成功后没有看到 `resp.Body.Close()`，长期运行可能影响连接复用与资源释放。

## 11. 文档维护索引

接口变化时，优先核对以下文件：

| 内容 | 源码位置 |
| --- | --- |
| 路由注册与中间件 | `backEnd/ioc/web.go` |
| 用户接口 | `backEnd/internal/web/user.go` |
| 文章接口 | `backEnd/internal/web/article.go` |
| 文章 DTO/VO | `backEnd/internal/web/article_vo.go` |
| 微信 OAuth2 | `backEnd/internal/web/wechat.go` |
| 统一响应包装 | `backEnd/pkg/ginx/wrapper.go` |
| JWT 实现 | `backEnd/internal/web/jwt/redis_jwt.go` |
| JWT 中间件 | `backEnd/internal/web/middleware/login_jwt.go` |
| 限流 | `backEnd/pkg/ginx/middleware/ratelimit/builder.go` |
| 指标入口 | `backEnd/internal/web/observerability.go`、`backEnd/main.go` |

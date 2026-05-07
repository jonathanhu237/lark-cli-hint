# FlowOps Partner Quota 排障 FAQ

适用范围：星桥科技 FlowOps 调度平台对外部合作方 API 的配额桶、限流窗口和补跑窗口管理。

## 现象

`flowctl check payment_settlement` 可能报 `FlowOpsRateLimitError`、HTTP 429、`partner_api_v2` vendor throttle、retry_after，或者批量结算接口在整点窗口被合作方限流并需要进入 `replay-window`。

## 可能原因

`payment_settlement` 使用合作方的 `partner_api_v2`。该接口按 quota bucket 控制结算批次，如果凌晨窗口的 batch settlement 请求已经耗尽配额，FlowOps 会把后续补跑放入 replay window，而不是让 DAG 直接无限重试。

终端里常见的是 429、throttle、retry_after；内部文档和群里更常说“配额桶”“partner quota”“补跑窗口”或“replay-window”。

## 推荐处理

1. 不要调高 DAG retry 次数，避免放大合作方限流。
2. 先申领补跑配额：`flowctl quota claim payment_settlement --bucket partner_api_v2 --mode replay-window`。
3. 重新执行 `flowctl check payment_settlement`。
4. 如果仍然 429，联系 `payments-core` 确认合作方当天的 quota bucket 是否已扩容。

## 证据线索

命中 `payment_settlement`、`partner_api_v2`、HTTP 429、vendor throttle、quota bucket、replay-window 或 batch settlement 时，优先按合作方配额路径处理。

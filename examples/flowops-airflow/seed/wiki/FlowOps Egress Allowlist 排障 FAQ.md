# FlowOps Egress Allowlist 排障 FAQ

适用范围：星桥科技 FlowOps 调度平台访问内部服务、服务端点审批和网络出口 profile。

## 现象

`flowctl check crm_sync` 报 `FlowOpsEgressDenied`、TLS handshake timeout、`crm.internal.svc:443` 无法访问，或者 customer delta pull 在默认出口 profile 下超时。

## 可能原因

`crm_sync` 从 CRM 内部服务拉取客户增量。默认 `default-deny` egress profile 不允许访问 `crm.internal.svc`，必须通过 FlowOps 网络出口白名单绑定到 `revenue-egress`。

这类问题在群里可能被描述为“服务端点没审批”“出口 profile 不对”“egress allowlist 缺失”，不一定直接说 network denied。

## 推荐处理

1. 不要在 DAG 中绕过 TLS 校验，也不要把服务地址改成公网代理。
2. 申请并绑定内部服务出口：`flowctl network allow crm_sync --service crm.internal.svc --profile revenue-egress`。
3. 重新执行 `flowctl check crm_sync`。
4. 如果仍超时，联系 `revenue-ops` 确认服务端点审批是否已经同步到 FlowOps 网关。

## 证据线索

命中 `crm_sync`、`crm.internal.svc`、FlowOpsEgressDenied、TLS handshake timeout、egress profile、allowlist、service endpoint approval 或 `revenue-egress` 时，优先按网络出口白名单路径处理。

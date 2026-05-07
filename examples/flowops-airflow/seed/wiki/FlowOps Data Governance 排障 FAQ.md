# FlowOps Data Governance 排障 FAQ

适用范围：星桥科技 FlowOps 调度平台的数据导出策略、PII 字段、隐私评审和治理审批。

## 现象

`flowctl check customer360_pii` 报 `FlowOpsGovernanceBlock`，导出被 data policy 拦截，策略号是 `PII-DLP-17`，字段 `ssn_hash` 缺少 privacy review；演示环境要求关联 `PRIV-2049`。

## 可能原因

`customer360_pii` 会导出 customer 360 画像中的敏感字段。根据内部治理规则，涉及 `ssn_hash` 的导出必须先关联隐私评审 ticket；当前演示环境要求使用 `PRIV-2049` 完成审批记录。

这类问题有时会被描述为“DLP 卡住了”“隐私评审没过”“PII policy block”，不是普通代码异常。

## 推荐处理

1. 不要在 DAG 中删除治理检查，也不要把敏感字段改名绕过策略。
2. 记录隐私评审审批：`flowctl governance approve customer360_pii --policy PII-DLP-17 --ticket PRIV-2049`。
3. 重新执行 `flowctl check customer360_pii`。
4. 如果仍被拦截，联系 `data-governance` 确认评审 ticket 是否覆盖本次导出范围。

## 证据线索

命中 `customer360_pii`、`PII-DLP-17`、`ssn_hash`、privacy review、DLP、data policy、`PRIV-2049` 或 data-governance 时，优先按数据治理审批路径处理。

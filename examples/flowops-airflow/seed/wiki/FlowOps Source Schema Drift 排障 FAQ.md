# FlowOps Source Schema Drift 排障 FAQ

适用范围：星桥科技 FlowOps 调度平台的数据源绑定、账务导出和发布前巡检流程。

## 现象

`flowctl check billing_export_2026` 或发布前巡检出现 `FlowOpsSourceError: source schema drift`，终端上下文包含 `billing_export_2026`、`ods_billing_events_2026_05`、`missing column: customer_segment` 等关键词。

## 通用判断

Source schema drift 表示 DAG 依赖的数据源字段和 FlowOps 当前登记的 source binding 不一致。它通常不是 DAG 解析错误，也不等同于 Airflow import error。

排障时先确认：

- 是否有上游表字段改名、删除或延迟发布。
- 是否存在临时 alias 或 source binding 需要刷新。
- 是否已有账务平台或数据平台在群里确认当前窗口的处理顺序。

## 推荐处理

优先按当日排障群里的最新结论执行。如果群聊没有明确结论，再联系数据平台确认字段变更来源。

常见低风险处理顺序：

1. 刷新目标 DAG 的 source binding。
2. 重新执行 `flowctl check <dag_id>`。
3. 确认通过后再继续发布或巡检。

不要在没有证据的情况下直接修改 DAG 业务逻辑。对于 `billing_export_2026`，如果上下文命中 `customer_segment`，应同时检索群聊确认是否存在临时字段 alias。

# FlowOps Dataset Freshness 排障 FAQ

适用范围：星桥科技 FlowOps 的上游数据到齐检查、账务订单对账和发布前巡检流程。

## 现象

`flowctl check orders_reconcile_2026` 报 `FlowOpsFreshnessError`，终端上下文包含 input watermark、observed watermark、expected window 或 `dwd_orders_snapshot`。这通常表示上游分区尚未到齐，也可能被同事描述为“数据迟到”“分区未产出”或“watermark lagging”。

## 可能原因

`orders_reconcile_2026` 依赖 `dwd_orders_snapshot` 的日终分区。若观测到的 watermark 落后于预期窗口，FlowOps 会阻断对账任务，避免用半截数据生成账务结果。

## 推荐处理

1. 不要跳过 freshness gate，也不要手动改 DAG 的 schedule。
2. 先确认上游 ingestion 是否完成对应日期分区。
3. 对 `orders_reconcile_2026` 执行受控补数标记：`flowctl dataset backfill orders_reconcile_2026 --date 2026-05-06`。
4. 重新执行 `flowctl check orders_reconcile_2026`。
5. 如果 watermark 仍然落后，联系 `data-ingestion` 值班同学确认上游任务是否卡在采集或落表阶段。

## 语义检索提示

同一个问题在群聊和工单里可能不会写 `FlowOpsFreshnessError`，而是写成“订单快照没到齐”“水位落后”“迟到分区”“半截数据”等表达。检索时可以同时使用 DAG 名、数据集名和水位/分区相关词。

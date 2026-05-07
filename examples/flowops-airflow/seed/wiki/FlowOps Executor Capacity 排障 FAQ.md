# FlowOps Executor Capacity 排障 FAQ

适用范围：星桥科技 FlowOps 调度平台的 worker slot starvation、pool saturation 和高扇出抽取任务排队。

## 现象

`flowctl check inventory_snapshot` 可能不会直接写“容量不足”，而是表现为任务长时间处于 queued state、没有 runnable slot、`bulk_export_pool` 被夜间 fanout export 占满，或者巡检提示 pool congestion。

## 可能原因

`inventory_snapshot` 是供应链侧的高扇出快照任务。它依赖 `bulk_export_pool` 的并发槽位；如果同一窗口里还有 nightly fanout exports，FlowOps 会把 DAG run 放在队列里等待，而不是立即失败。

这类问题在复盘里也叫 worker 饥饿、slot starvation、pool saturation，和 DAG 代码本身通常无关。

## 推荐处理

1. 先确认是否只有 `inventory_snapshot` 被排队，避免误判为全局调度器故障。
2. 对演示环境执行：`flowctl capacity bump inventory_snapshot --pool bulk_export_pool --slots 4`。
3. 重新执行 `flowctl check inventory_snapshot`。
4. 如果生产环境仍拥塞，联系 `supply-chain` 和 FlowOps 平台值班调整该窗口的 pool 配额。

## 证据线索

命中 `inventory_snapshot`、`bulk_export_pool`、queued state、runnable slot、fanout export、slot starvation 或 pool saturation 时，优先走执行容量排查路线。

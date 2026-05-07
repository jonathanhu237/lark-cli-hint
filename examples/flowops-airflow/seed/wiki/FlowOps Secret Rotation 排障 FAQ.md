# FlowOps Secret Rotation 排障 FAQ

适用范围：星桥科技 FlowOps 的数仓访问凭证、广告投放数据抽取和轮换窗口后的巡检。

## 现象

`flowctl check ad_spend_daily` 报 `FlowOpsWarehouseAuthError`、warehouse session broker 拒绝连接、credential lease stale、secret epoch 落后，或者抽取任务无法打开 ads mart snapshot。

## 可能原因

广告数据 DAG 使用的 `warehouse_token` 没有跟上本周密钥轮换。FlowOps 会比较本地 secret epoch 与平台要求的 epoch；如果任务仍持有旧 epoch，数仓 broker 会拒绝发放 session。

这个问题在群聊里也常被描述为“仓库 token 旧了”“凭证租约过期”“轮换后没刷新 secret”，不一定直接写 `FlowOpsWarehouseAuthError`。

## 推荐处理

1. 不要把 token 写进 DAG 文件，也不要把旧 token 复制到本地配置。
2. 按目标 epoch 更新 FlowOps secret：`flowctl secrets renew ad_spend_daily --secret warehouse_token --epoch 2026w19`。
3. 重新执行 `flowctl check ad_spend_daily`。
4. 如果仍失败，联系 `ads-analytics` 和平台值班确认本周 epoch 是否已经在密钥中心发布。

## 证据线索

命中 `ad_spend_daily`、`warehouse_token`、`credential lease`、`secret rotation`、`epoch=2026w18`、`required epoch=2026w19` 时，优先按密钥轮换路径排查。

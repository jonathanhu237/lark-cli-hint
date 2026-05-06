适用对象：星桥科技所有提交到 FlowOps 的 DAG。

# 解析期规则

- DAG 文件顶层不得调用 `Variable.get`、连接读取、数据库查询或 HTTP 请求。
- DAG 文件顶层不得依赖只存在于某个环境的配置。
- DAG import 必须在本地、预发和生产环境保持稳定。

# 推荐写法

- 在任务函数内部读取 `billing_region` 等运行时配置。
- 对可选配置提供显式默认值，并在任务日志中记录最终值。
- 对必填配置，在任务开始后抛出带修复建议的错误。

# 发布前检查

```sh
flowctl check billing_daily
```

如果出现 import error，先修复 DAG 解析问题，再继续发布。不要通过跳过巡检或直接重启调度器绕过。

# lark-cue 检索提示

当终端出现 `DAG import error`、`billing_daily`、`Variable.get`、`billing_region` 等关键词时，优先检索 FlowOps FAQ、历史故障复盘和本规范。

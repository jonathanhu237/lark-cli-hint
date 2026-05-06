适用范围：星桥科技 FlowOps 调度平台的 DAG 发布、巡检和本地复现流程。

# 现象

`flowctl check billing_daily` 或发布前巡检出现 DAG import error，终端上下文包含 `billing_daily.py`、`Variable.get("billing_region")`、`billing_region` 不存在等关键词。

# 可能原因

`billing_daily` 在 DAG 文件解析阶段读取 FlowOps Variable。解析阶段运行在调度器、Webserver 和 CLI 的 DAG import 流程里，如果目标环境尚未配置 `billing_region`，DAG 文件会直接导入失败，任务还没有开始运行。

# 推荐处理

优先把 `Variable.get("billing_region")` 移到任务运行时或带默认值的配置加载函数里。若正在处理线上阻塞，可以先在目标 FlowOps 环境补齐 `billing_region` 变量作为短期解法，然后补提交代码修复解析期读取。

# 证据线索

- 内部规则：DAG 顶层代码不得依赖环境内 Variable、连接或外部服务。
- 排障关键词：`DAG import error`、`billing_daily`、`billing_region`、`Variable.get`、`parse time`。
- 责任团队：星桥科技 FlowOps 平台组。

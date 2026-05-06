复盘编号：FLOWOPS-INC-2026-0418

# 背景

账务团队发布 `billing_daily` 后，FlowOps 巡检在预发环境发现 DAG import error。调度器没有创建当日账务聚合任务，发布窗口被迫暂停。

# 影响

- 预发账务聚合延迟 28 分钟。
- 发布负责人需要在 FlowOps 控制台和终端之间来回核对错误。
- 事后确认业务数据未丢失。

# 根因

DAG 文件顶层执行了 `Variable.get("billing_region")`。预发环境没有该变量，Airflow 在解析 DAG 时抛错，导致 `billing_daily` 无法被识别为可调度 DAG。

# 修复结论

1. 顶层只允许声明 DAG、任务和静态常量。
2. 环境变量、FlowOps Variable、连接信息必须在任务运行时读取。
3. 发布前必须执行 `flowctl check <dag_id>`，并把 import error 作为阻断项。

# 后续动作

账务团队需要把 region 读取封装进任务函数，并在失败提示里输出当前环境名和缺失变量名。

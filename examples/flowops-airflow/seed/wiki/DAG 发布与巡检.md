本栏目沉淀 FlowOps DAG 从本地开发到发布前巡检的内部规则。

# 发布前必须确认

- DAG 文件可以被 Airflow 和 FlowOps CLI 稳定解析。
- 顶层代码不读取只存在于某个环境的变量、连接或外部服务。
- `flowctl check <dag_id>` 没有 import error、依赖缺失或配置读取异常。

如果巡检失败，先处理解析期问题，再继续提交发布审批。

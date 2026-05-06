from __future__ import annotations

from datetime import datetime

from airflow import DAG
from airflow.models import Variable
from airflow.operators.empty import EmptyOperator
from airflow.operators.python import PythonOperator


BILLING_REGION_HELP = (
    "Missing required Airflow Variable 'billing_region'. "
    "Set it with: flowctl airflow variables set billing_region <region>, "
    "for example: flowctl airflow variables set billing_region cn-north."
)


def validate_billing_region() -> None:
    """Read required billing config at task runtime, not DAG parse time."""
    try:
        Variable.get("billing_region")
    except KeyError as exc:
        raise RuntimeError(BILLING_REGION_HELP) from exc


with DAG(
    dag_id="billing_daily",
    description="星桥科技 FlowOps billing aggregation pipeline",
    schedule="@daily",
    start_date=datetime(2026, 1, 1),
    catchup=False,
    tags=["flowops", "billing"],
) as dag:
    validate_region = PythonOperator(
        task_id="validate_billing_region",
        python_callable=validate_billing_region,
    )
    export_billing_snapshot = EmptyOperator(task_id="export_billing_snapshot")

    validate_region >> export_billing_snapshot

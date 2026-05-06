from __future__ import annotations

from datetime import datetime

from airflow import DAG
from airflow.models import Variable
from airflow.operators.empty import EmptyOperator


# FlowOps intentionally keeps this broken for the lark-cue demo:
# reading Variables at module import time makes DAG parsing fail when the
# variable is absent from the Airflow metadata DB.
BILLING_REGION = Variable.get("billing_region")


with DAG(
    dag_id="billing_daily",
    description="星桥科技 FlowOps billing aggregation pipeline",
    schedule="@daily",
    start_date=datetime(2026, 1, 1),
    catchup=False,
    tags=["flowops", "billing", BILLING_REGION],
) as dag:
    validate_region = EmptyOperator(task_id=f"validate_{BILLING_REGION}")
    export_billing_snapshot = EmptyOperator(task_id="export_billing_snapshot")

    validate_region >> export_billing_snapshot

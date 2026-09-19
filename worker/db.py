"""
Postgres persistence for job history.

Mirrors the API's writes: when a job is created, the API writes to
Postgres. When a job completes or fails, the worker updates that row.

Redis remains the source of truth for the live queue and state.
Postgres is the durable record of the final outcome.
"""
import os
from typing import Any, Dict, Optional

import psycopg


_connection_url: Optional[str] = None


def _get_url() -> Optional[str]:
    """Read DATABASE_URL lazily so it's picked up after env is set."""
    global _connection_url
    if _connection_url is None:
        _connection_url = os.getenv("DATABASE_URL", "")
    return _connection_url or None


def _connect():
    """Open a new short-lived connection to Postgres."""
    url = _get_url()
    if not url:
        return None
    return psycopg.connect(url, autocommit=True)


def is_enabled() -> bool:
    """Return True if Postgres is configured."""
    return bool(_get_url())


def mark_running(job_id: str, solver_type: str) -> None:
    """
    Update a job row to status='running' and record the solver.
    Called when a worker picks up a job.
    """
    conn = _connect()
    if conn is None:
        return

    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                UPDATE jobs
                SET status = %s, solver_type = %s
                WHERE id = %s
                """,
                ("running", solver_type, job_id),
            )
    except Exception as e:
        print(f"  ! Postgres mark_running failed for {job_id}: {e}")
    finally:
        conn.close()


def mark_completed(job_id: str, result: Dict[str, Any]) -> None:
    """
    Write the final result of a completed job.

    Expected keys in `result`:
      - best_fitness: float
      - best_individual: list | dict
      - history: list
      - total_generations: int
      - result_meta: dict (optional)
    """
    conn = _connect()
    if conn is None:
        return

    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                UPDATE jobs
                SET status = 'completed',
                    best_fitness = %s,
                    best_individual = %s,
                    history = %s,
                    total_generations = %s,
                    result_meta = %s,
                    completed_at = NOW()
                WHERE id = %s
                """,
                (
                    result.get("best_fitness"),
                    psycopg.types.json.Json(result.get("best_individual", [])),
                    psycopg.types.json.Json(result.get("history", [])),
                    result.get("total_generations"),
                    psycopg.types.json.Json(result.get("result_meta", {})),
                    job_id,
                ),
            )
    except Exception as e:
        print(f"  ! Postgres mark_completed failed for {job_id}: {e}")
    finally:
        conn.close()


def mark_failed(job_id: str, error_message: str) -> None:
    """Record a failed job with its error message."""
    conn = _connect()
    if conn is None:
        return

    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                UPDATE jobs
                SET status = 'failed', error = %s
                WHERE id = %s
                """,
                (error_message, job_id),
            )
    except Exception as e:
        print(f"  ! Postgres mark_failed failed for {job_id}: {e}")
    finally:
        conn.close()


def health_check() -> bool:
    """Return True if Postgres is reachable."""
    conn = _connect()
    if conn is None:
        return False
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            cur.fetchone()
        return True
    except Exception:
        return False
    finally:
        conn.close()
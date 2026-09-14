"""
Formats benchmark results as CSV and markdown.
"""
import csv
import statistics
from datetime import datetime
from pathlib import Path
from typing import Dict, List


def write_csv(results: List[Dict], path: Path) -> None:
    """Write results to a CSV file."""
    path.parent.mkdir(parents=True, exist_ok=True)
    fields = ["id", "difficulty", "num_nurses", "num_days", "status",
              "solver_status", "objective", "solve_time", "error"]
    
    with path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fields, extrasaction="ignore")
        writer.writeheader()
        for r in results:
            writer.writerow(r)


def write_markdown(results: List[Dict], path: Path, label: str = "CP-SAT") -> None:
    """Write a markdown report with summary + table."""
    path.parent.mkdir(parents=True, exist_ok=True)
    
    ok = [r for r in results if r["status"] == "ok"]
    errors = [r for r in results if r["status"] != "ok"]
    
    # Group by difficulty
    by_difficulty = {}
    for r in ok:
        by_difficulty.setdefault(r["difficulty"], []).append(r)
    
    # Build report
    lines = []
    lines.append(f"# {label} Benchmark Report\n")
    lines.append(f"*Generated: {datetime.utcnow().isoformat()}Z*\n")
    lines.append(f"**Total instances:** {len(results)}  ")
    lines.append(f"**Succeeded:** {len(ok)}  ")
    lines.append(f"**Failed:** {len(errors)}\n")
    
    # Summary table by difficulty
    lines.append("## Summary by difficulty\n")
    lines.append("| Difficulty | Count | Mean solve time | Median solve time | Max solve time | Optimal count |")
    lines.append("| :--- | ---: | ---: | ---: | ---: | ---: |")
    
    for difficulty in ["small", "medium", "large"]:
        group = by_difficulty.get(difficulty, [])
        if not group:
            continue
        times = [r["solve_time"] for r in group]
        optimal = sum(1 for r in group if r["solver_status"] == "OPTIMAL")
        lines.append(
            f"| {difficulty} | {len(group)} | "
            f"{statistics.mean(times):.3f}s | "
            f"{statistics.median(times):.3f}s | "
            f"{max(times):.3f}s | "
            f"{optimal}/{len(group)} |"
        )
    lines.append("")
    
    # Full results table
    lines.append("## All instances\n")
    lines.append("| ID | Difficulty | Size | Status | Objective | Solve time |")
    lines.append("| :--- | :--- | :--- | :--- | ---: | ---: |")
    
    for r in results:
        size = f"{r['num_nurses']}n × {r['num_days']}d"
        obj = r["objective"] if r["objective"] is not None else "—"
        status = r["solver_status"] if r["status"] == "ok" else "ERROR"
        lines.append(
            f"| {r['id']} | {r['difficulty']} | {size} | "
            f"{status} | {obj} | {r['solve_time']}s |"
        )
    lines.append("")
    
    # Errors section
    if errors:
        lines.append("## Errors\n")
        for r in errors:
            lines.append(f"- **{r['id']}**: {r['error']}")
        lines.append("")
    
    # Notes
    lines.append("## Notes\n")
    lines.append("- All results are single-run; no warm-up or repetition.")
    lines.append("- Solve time includes model construction and solver execution.")
    lines.append("- `OPTIMAL` means the solver proved no better solution exists.")
    lines.append("- `FEASIBLE` means a solution was found but not proven optimal.\n")
    
    path.write_text("\n".join(lines))


def print_console_summary(results: List[Dict]) -> None:
    """Print a short summary to stdout."""
    ok = [r for r in results if r["status"] == "ok"]
    if not ok:
        print("\nNo successful instances.")
        return
    
    print("\n" + "=" * 60)
    print("BENCHMARK SUMMARY")
    print("=" * 60)
    for difficulty in ["small", "medium", "large"]:
        group = [r for r in ok if r["difficulty"] == difficulty]
        if not group:
            continue
        times = [r["solve_time"] for r in group]
        print(f"  {difficulty:8s}: {len(group):2d} instances | "
              f"mean={statistics.mean(times):.3f}s | "
              f"max={max(times):.3f}s")
    print("=" * 60)
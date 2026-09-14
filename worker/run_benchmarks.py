"""
Run EvoFarm benchmarks.

Usage:
  cd worker
  python3 run_benchmarks.py
  
  # Or filter:
  python3 run_benchmarks.py --difficulty small
  python3 run_benchmarks.py --time-limit 10
"""
import argparse
import sys
from pathlib import Path

# Add worker/ to path so imports work
sys.path.insert(0, str(Path(__file__).resolve().parent))

from benchmarks import instances as inst_mod
from benchmarks import runner, report


def main():
    parser = argparse.ArgumentParser(description="Run EvoFarm benchmarks")
    parser.add_argument("--difficulty", choices=["small", "medium", "large", "all"],
                        default="all", help="Which instances to run")
    parser.add_argument("--time-limit", type=float, default=60.0,
                        help="CP-SAT time limit per instance (seconds)")
    parser.add_argument("--output-dir", type=str,
                        default="../docs/benchmarks",
                        help="Where to write results")
    parser.add_argument("--label", type=str, default="cpsat-baseline",
                        help="Label for output files")
    args = parser.parse_args()
    
    # Pick instances
    if args.difficulty == "all":
        instances = inst_mod.all_instances()
    elif args.difficulty == "small":
        instances = inst_mod.small_instances()
    elif args.difficulty == "medium":
        instances = inst_mod.medium_instances()
    elif args.difficulty == "large":
        instances = inst_mod.large_instances()
    else:
        instances = inst_mod.all_instances()
    
    print(f"Running {len(instances)} instances "
          f"(difficulty={args.difficulty}, time_limit={args.time_limit}s)\n")
    
    # Run
    results = runner.run_benchmark(instances, time_limit=args.time_limit)
    
    # Write outputs
    output_dir = Path(args.output_dir)
    csv_path = output_dir / f"results-{args.label}.csv"
    md_path = output_dir / f"{args.label}.md"
    
    report.write_csv(results, csv_path)
    report.write_markdown(results, md_path, label=args.label.replace("-", " ").title())
    report.print_console_summary(results)
    
    print(f"\nWrote: {csv_path}")
    print(f"Wrote: {md_path}")


if __name__ == "__main__":
    main()
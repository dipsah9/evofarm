#!/bin/bash
JOB_ID=$1
if [ -z "$JOB_ID" ]; then
    echo "Usage: ./watch-job.sh <job_id>"
    exit 1
fi

while true; do
    clear
    echo "Monitoring Job: $JOB_ID"
    echo "================================"
    curl -s http://localhost:8080/jobs/$JOB_ID | python3 -c "
import sys, json
data = json.load(sys.stdin)
status = data.get('status', 'unknown')
fitness = data.get('best_fitness', 0)
progress = float(data.get('progress', 0))
bars = int(progress * 50)
print(f'Status: {status}')
print(f'Progress: [{\"#\"*bars}{\".\"*(50-bars)}] {progress*100:.1f}%')
print(f'Best Fitness: {fitness:.4f}')
if status == 'completed':
    print('✅ Job complete!')
elif status == 'failed':
    print('❌ Job failed:', data.get('error', ''))
"
    sleep 2
done
FROM python:3.11-slim

WORKDIR /app

ENV PYTHONUNBUFFERED=1

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py .
COPY routing.py .
COPY db.py .
COPY solvers ./solvers
COPY environments ./environments
COPY problems ./problems
COPY benchmarks ./benchmarks

CMD ["python", "-u", "main.py"]
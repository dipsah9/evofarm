package main

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "log"
    "net/http"
	"os"
    "strconv"

    "github.com/go-redis/redis/v8"
    "github.com/gorilla/mux"
)

var ctx = context.Background()
var rdb *redis.Client

type JobRequest struct {
    PopulationSize int    `json:"population_size"`
    Generations    int    `json:"generations"`
    FitnessFunction string `json:"fitness_function"`
}

type JobStatus struct {
    JobID         string  `json:"job_id"`
    Status        string  `json:"status"`
    BestFitness   float64 `json:"best_fitness"`
    Progress      float64 `json:"progress"`
    BestIndividual []float64 `json:"best_individual,omitempty"`
    Error         string  `json:"error,omitempty"`
}

func main() {
    // Connect to Redis
    rdb = redis.NewClient(&redis.Options{
        Addr: getEnv("REDIS_ADDR", "localhost:6379"),
    })

    // Test connection
    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Fatal("Could not connect to Redis:", err)
    }
    log.Println("Connected to Redis")

    // Setup router
    r := mux.NewRouter()
    r.HandleFunc("/jobs", submitJob).Methods("POST")
    r.HandleFunc("/jobs/{id}", getJobStatus).Methods("GET")
    r.HandleFunc("/health", healthCheck).Methods("GET")

    log.Println("API listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}

func submitJob(w http.ResponseWriter, r *http.Request) {
    var req JobRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Validate
    if req.PopulationSize == 0 {
        req.PopulationSize = 100
    }
    if req.Generations == 0 {
        req.Generations = 50
    }
    if req.FitnessFunction == "" {
        req.FitnessFunction = "xor"
    }

    // Generate unique job ID
    bytes := make([]byte, 16)
    rand.Read(bytes)
    jobID := hex.EncodeToString(bytes)

    // Create job data
    job := JobStatus{
        JobID:   jobID,
        Status:  "pending",
        BestFitness: 0,
        Progress: 0,
    }

    // Store job in Redis as hash
    jobData := map[string]interface{}{
        "status":          job.Status,
        "best_fitness":    job.BestFitness,
        "progress":        job.Progress,
        "population_size": req.PopulationSize,
        "generations":     req.Generations,
        "fitness_function": req.FitnessFunction,
        "best_individual": "[]",
        "error":           "",
    }
    
    err := rdb.HSet(ctx, "job:"+jobID, jobData).Err()
    if err != nil {
        http.Error(w, "Failed to store job", http.StatusInternalServerError)
        return
    }

    // Push to queue for worker
    err = rdb.LPush(ctx, "job_queue", jobID).Err()
    if err != nil {
        http.Error(w, "Failed to queue job", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"job_id": jobID, "status": "pending"})
}

func getJobStatus(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    jobID := vars["id"]

    // Get job data from Redis
    data, err := rdb.HGetAll(ctx, "job:"+jobID).Result()
    if err != nil || len(data) == 0 {
        http.Error(w, "Job not found", http.StatusNotFound)
        return
    }

    bestFitness, _ := strconv.ParseFloat(data["best_fitness"], 64)
    progress, _ := strconv.ParseFloat(data["progress"], 64)

    job := JobStatus{
        JobID:         jobID,
        Status:        data["status"],
        BestFitness:   bestFitness,
        Progress:      progress,
        BestIndividual: []float64{},
        Error:         data["error"],
    }

    // Parse best_individual if it exists and job is completed
    if data["best_individual"] != "" && data["best_individual"] != "[]" {
        var weights []float64
        json.Unmarshal([]byte(data["best_individual"]), &weights)
        job.BestIndividual = weights
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(job)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func getEnv(key, defaultValue string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return defaultValue
}
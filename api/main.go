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


func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Allow requests from any origin (for development)
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
        w.Header().Set("Access-Control-Max-Age", "3600")

        // Handle preflight OPTIONS request
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}

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
	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(r)))
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

    data, err := rdb.HGetAll(ctx, "job:"+jobID).Result()
    if err != nil || len(data) == 0 {
        http.Error(w, "Job not found", http.StatusNotFound)
        return
    }

    bestFitness, _ := strconv.ParseFloat(data["best_fitness"], 64)
    progress, _ := strconv.ParseFloat(data["progress"], 64)
    totalGenerations, _ := strconv.Atoi(data["total_generations"])

    // Parse history
    var history []map[string]interface{}
    if data["history"] != "" {
        json.Unmarshal([]byte(data["history"]), &history)
    }

    var bestIndividual []float64
    if data["best_individual"] != "" && data["best_individual"] != "[]" {
        json.Unmarshal([]byte(data["best_individual"]), &bestIndividual)
    }

    response := map[string]interface{}{
        "job_id":           jobID,
        "status":           data["status"],
        "best_fitness":     bestFitness,
        "progress":         progress,
        "best_individual":  bestIndividual,
        "history":          history,
        "total_generations": totalGenerations,
        "error":            data["error"],
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
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
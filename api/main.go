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
    SolverType      string `json:"solver_type"`
    Problem         string  `json:"problem"`
    TimeLimitSeconds float64 `json:"time_limit_seconds"`
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
    // rdb = redis.NewClient(&redis.Options{
    //     Addr: getEnv("REDIS_ADDR", "localhost:6379"),
    // })

	redisURL := getEnv("REDIS_URL", "")
	if redisURL == "" {
		// Fallback for local development
		redisURL = "redis://" + getEnv("REDIS_ADDR", "localhost:6379")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Invalid REDIS_URL:", err)
	}

	rdb = redis.NewClient(opt)

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
	r.HandleFunc("/problems", listProblems).Methods("GET")

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
    
	// If user provides a problem but no solver, leave solver empty —
	// the worker will route. This is the new preferred flow.
	// If user provides neither, fall back to evolution+xor for backward compat.
	if req.SolverType == "" && req.Problem == "" {
		req.SolverType = "evolution"
		req.FitnessFunction = "xor"
	}
	if req.TimeLimitSeconds == 0 {
		req.TimeLimitSeconds = 30
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
        "solver_type":      req.SolverType,
        "problem":          req.Problem,
        "time_limit_seconds": req.TimeLimitSeconds,
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

    var bestIndividual interface{}
	if data["best_individual"] != "" && data["best_individual"] != "[]" {
		if err := json.Unmarshal([]byte(data["best_individual"]), &bestIndividual); err != nil {
			log.Printf("Warning: could not parse best_individual: %v", err)
			bestIndividual = []interface{}{}
		}
	} else {
		bestIndividual = []interface{}{}
	}

    // Parse result metadata if present.
    var resultMeta interface{}
    if data["result_meta"] != "" {
        if err := json.Unmarshal([]byte(data["result_meta"]), &resultMeta); err != nil {
            resultMeta = map[string]interface{}{}
        }
    } else {
        resultMeta = map[string]interface{}{}
    }

    response := map[string]interface{}{
        "job_id":           jobID,
        "status":           data["status"],
        "best_fitness":     bestFitness,
        "progress":         progress,
        "best_individual":  bestIndividual,
        "history":          history,
        "total_generations": totalGenerations,
        "result_meta":      resultMeta,
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

func listProblems(w http.ResponseWriter, r *http.Request) {
    // Hardcoded catalog. Must stay in sync with worker/routing.py.
    // When this drifts, we'll move to auto-discovery.
    catalog := map[string]interface{}{
        "xor": map[string]interface{}{
            "solver":      "evolution",
            "description": "Evolve a neural network to solve the XOR truth table",
            "category":    "puzzle",
        },
        "portfolio": map[string]interface{}{
            "solver":      "evolution",
            "description": "Maximize Sharpe ratio of a multi-asset portfolio",
            "category":    "optimization",
        },
        "nurse_rostering": map[string]interface{}{
            "solver":      "cpsat",
            "description": "Assign nurses to shifts satisfying coverage and rest rules",
            "category":    "scheduling",
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(catalog)
}
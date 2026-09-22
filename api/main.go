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
var db *DB

type JobRequest struct {
    PopulationSize int    `json:"population_size"`
    Generations    int    `json:"generations"`
    FitnessFunction string `json:"fitness_function"`
    SolverType      string `json:"solver_type"`
    Problem         string  `json:"problem"`
	Config           map[string]interface{} `json:"config"`
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


	// Connect to Postgres (Neon)
	dbURL := getEnv("DATABASE_URL", "")
	if dbURL != "" {
		var err error
		db, err = NewDB(ctx, dbURL)
		if err != nil {
			log.Fatal("Could not connect to Postgres:", err)
		}
		log.Println("Connected to Postgres")
	} else {
		log.Println("WARNING: DATABASE_URL not set — Postgres persistence disabled")
	}

    // Test connection
    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Fatal("Could not connect to Redis:", err)
    }
    log.Println("Connected to Redis")

    rateCfg := LoadRateLimitConfig()
	log.Printf("Rate limit: %d requests per %s per IP", rateCfg.MaxRequests, rateCfg.Window)

	r := buildRouter(rateCfg)

	log.Println("API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func buildRouter(rateCfg RateLimitConfig) http.Handler {
	r := mux.NewRouter()
	r.Use(corsMiddleware)

	// Explicit OPTIONS handler for CORS preflight.
	// Gorilla mux returns 405 for unmatched OPTIONS unless we
	// register a route that handles every path.
	r.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Public routes
	r.HandleFunc("/auth/register", registerHandler).Methods("POST")
	r.HandleFunc("/auth/login", loginHandler).Methods("POST")
	r.HandleFunc("/health", healthCheck).Methods("GET")
	r.HandleFunc("/problems", listProblems).Methods("GET")

	// Authenticated routes
	r.HandleFunc("/auth/me", meHandler).Methods("GET")
	r.HandleFunc("/jobs/history", authMiddleware(http.HandlerFunc(listJobHistory)).ServeHTTP).Methods("GET")
	r.HandleFunc("/jobs/{id}", authMiddleware(http.HandlerFunc(getJobStatus)).ServeHTTP).Methods("GET")
	r.HandleFunc("/jobs", authMiddleware(rateLimitMiddleware(rateCfg)(http.HandlerFunc(submitJob))).ServeHTTP).Methods("POST")
	r.HandleFunc("/rate-limit-status", authMiddleware(http.HandlerFunc(rateLimitStatusHandler(rateCfg))).ServeHTTP).Methods("GET")

	return r
}

func submitJob(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

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
	if req.FitnessFunction == "" && req.Problem == "" {
		req.FitnessFunction = "xor"
	}

	// Generate unique job ID
	bytes := make([]byte, 16)
	rand.Read(bytes)
	jobID := hex.EncodeToString(bytes)

	// Create job data
	job := JobStatus{
		JobID:       jobID,
		Status:      "pending",
		BestFitness: 0,
		Progress:    0,
	}

	// Convert config to JSON string (for CP-SAT problems)
	configJSON := "{}"
	if req.Config != nil {
		if bytes, err := json.Marshal(req.Config); err == nil {
			configJSON = string(bytes)
		}
	}

	// Store job in Redis as hash
	jobData := map[string]interface{}{
		"status":             job.Status,
		"best_fitness":       job.BestFitness,
		"progress":           job.Progress,
		"population_size":    req.PopulationSize,
		"generations":        req.Generations,
		"fitness_function":   req.FitnessFunction,
		"solver_type":        req.SolverType,
		"problem":            req.Problem,
		"config":             configJSON,
		"time_limit_seconds": req.TimeLimitSeconds,
		"user_id":            userID,
		"best_individual":    "[]",
		"error":              "",
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

	// Also write to Postgres for durability (fire-and-forget)
	if db != nil {
		if err := db.CreateJob(ctx, JobRecord{
			ID:         jobID,
			UserID:     &userID,
			Problem:    req.Problem,
			SolverType: req.SolverType,
			Config: map[string]interface{}{
				"population_size":    req.PopulationSize,
				"generations":        req.Generations,
				"fitness_function":   req.FitnessFunction,
				"time_limit_seconds": req.TimeLimitSeconds,
			},
			Status: "pending",
		}); err != nil {
			log.Printf("WARN: could not write job %s to Postgres: %v", jobID, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"job_id": jobID, "status": "pending"})
}

func getJobStatus(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	jobID := vars["id"]

	// Try Redis first (fast path — live jobs)
	data, err := rdb.HGetAll(ctx, "job:"+jobID).Result()
	if err == nil && len(data) > 0 {
		// Verify ownership — if the job has a user_id, it must match.
		// Jobs without a user_id (legacy) are visible to any authenticated caller.
		jobOwner := data["user_id"]
		if jobOwner != "" && jobOwner != userID {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		writeJobFromRedis(w, jobID, data)
		return
	}

	// Fall back to Postgres (durable history)
	if db != nil {
		job, err := db.GetJobForUser(ctx, jobID, userID)
		if err == nil && job != nil {
			writeJobFromPostgres(w, job)
			return
		}
	}

	http.Error(w, "job not found", http.StatusNotFound)
}

// writeJobFromRedis renders a job from a Redis hash map.
func writeJobFromRedis(w http.ResponseWriter, jobID string, data map[string]string) {
	bestFitness, _ := strconv.ParseFloat(data["best_fitness"], 64)
	progress, _ := strconv.ParseFloat(data["progress"], 64)
	totalGenerations, _ := strconv.Atoi(data["total_generations"])

	var history []map[string]interface{}
	if data["history"] != "" {
		json.Unmarshal([]byte(data["history"]), &history)
	}

	var bestIndividual interface{}
	if data["best_individual"] != "" && data["best_individual"] != "[]" {
		if err := json.Unmarshal([]byte(data["best_individual"]), &bestIndividual); err != nil {
			bestIndividual = []interface{}{}
		}
	} else {
		bestIndividual = []interface{}{}
	}

	var resultMeta interface{}
	if data["result_meta"] != "" {
		if err := json.Unmarshal([]byte(data["result_meta"]), &resultMeta); err != nil {
			resultMeta = map[string]interface{}{}
		}
	} else {
		resultMeta = map[string]interface{}{}
	}

	response := map[string]interface{}{
		"job_id":            jobID,
		"status":            data["status"],
		"best_fitness":      bestFitness,
		"progress":          progress,
		"best_individual":   bestIndividual,
		"history":           history,
		"total_generations": totalGenerations,
		"result_meta":       resultMeta,
		"error":             data["error"],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeJobFromPostgres renders a job from a Postgres record.
func writeJobFromPostgres(w http.ResponseWriter, job *JobRecord) {
	// Completed jobs from Postgres are always 100% done
	var progress float64 = 1.0
	if job.Status != "completed" && job.Status != "failed" {
		progress = 0.5 // running or pending
	}

	response := map[string]interface{}{
		"job_id":            job.ID,
		"status":            job.Status,
		"best_fitness":      job.BestFitness,
		"progress":          progress,
		"best_individual":   job.BestIndividual,
		"history":           job.History,
		"total_generations": job.TotalGenerations,
		"result_meta":       job.ResultMeta,
		"error":             job.Error,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func listJobHistory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	if db == nil {
		http.Error(w, "History not available (Postgres not configured)",
			http.StatusServiceUnavailable)
		return
	}

	// Parse limit (default 50, max 200)
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
			if limit > 200 {
				limit = 200
			}
		}
	}

	jobs, err := db.ListJobs(ctx, userID, limit)
	if err != nil {
		log.Printf("Error listing jobs: %v", err)
		http.Error(w, "Failed to list jobs", http.StatusInternalServerError)
		return
	}

	// Convert to a stable JSON shape
	type JobSummary struct {
		ID               string   `json:"id"`
		Problem          string   `json:"problem"`
		SolverType       string   `json:"solver_type"`
		Status           string   `json:"status"`
		BestFitness      *float64 `json:"best_fitness"`
		TotalGenerations *int     `json:"total_generations"`
	}

	out := make([]JobSummary, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, JobSummary{
			ID:               j.ID,
			Problem:          j.Problem,
			SolverType:       j.SolverType,
			Status:           j.Status,
			BestFitness:      j.BestFitness,
			TotalGenerations: j.TotalGenerations,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
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
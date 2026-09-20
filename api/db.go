package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a Postgres connection pool with typed job operations.
type DB struct {
	pool *pgxpool.Pool
}

// JobRecord is the durable representation of a job.
// Matches the schema in docs/schema.sql.
type JobRecord struct {
	ID               string
	UserID           *string
	Problem          string
	SolverType       string
	Config           map[string]interface{}
	Status           string
	BestFitness      *float64
	BestIndividual   interface{}
	History          interface{}
	TotalGenerations *int
	ResultMeta       map[string]interface{}
	Error            *string
}

// NewDB connects to Postgres and verifies the connection.
func NewDB(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{pool: pool}, nil
}

// Close releases the connection pool.
func (d *DB) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

// CreateJob inserts a new job row with the owner's user_id.
// user_id is required.
func (d *DB) CreateJob(ctx context.Context, j JobRecord) error {
	if j.UserID == nil || *j.UserID == "" {
		return fmt.Errorf("CreateJob: user_id is required")
	}
	configJSON, _ := json.Marshal(j.Config)
	_, err := d.pool.Exec(ctx, `
		INSERT INTO jobs (id, user_id, problem, solver_type, config, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, j.ID, j.UserID, j.Problem, j.SolverType, configJSON, j.Status)
	return err
}

// UpdateJobCompleted writes the final result of a completed job.
func (d *DB) UpdateJobCompleted(ctx context.Context, j JobRecord) error {
	bestIndJSON, _ := json.Marshal(j.BestIndividual)
	historyJSON, _ := json.Marshal(j.History)
	metaJSON, _ := json.Marshal(j.ResultMeta)

	_, err := d.pool.Exec(ctx, `
		UPDATE jobs
		SET status = $1,
		    best_fitness = $2,
		    best_individual = $3,
		    history = $4,
		    total_generations = $5,
		    result_meta = $6,
		    completed_at = NOW()
		WHERE id = $7
	`, j.Status, j.BestFitness, bestIndJSON, historyJSON,
		j.TotalGenerations, metaJSON, j.ID)
	return err
}

// UpdateJobFailed marks a job as failed with an error message.
func (d *DB) UpdateJobFailed(ctx context.Context, id, errMsg string) error {
	_, err := d.pool.Exec(ctx, `
		UPDATE jobs SET status = 'failed', error = $1 WHERE id = $2
	`, errMsg, id)
	return err
}

// ListJobs returns the most recent jobs for a given user.
func (d *DB) ListJobs(ctx context.Context, userID string, limit int) ([]JobRecord, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, problem, solver_type, status, best_fitness,
		       total_generations, created_at, completed_at
		FROM jobs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobRecord
	for rows.Next() {
		var j JobRecord
		var createdAt time.Time
		var completedAt *time.Time
		if err := rows.Scan(&j.ID, &j.Problem, &j.SolverType, &j.Status,
			&j.BestFitness, &j.TotalGenerations, &createdAt, &completedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// GetJob returns a single job from Postgres by ID.
func (d *DB) GetJob(ctx context.Context, id string) (*JobRecord, error) {
	var j JobRecord
	var configJSON, bestIndJSON, historyJSON, metaJSON []byte
	var createdAt, updatedAt time.Time
	var completedAt *time.Time

	err := d.pool.QueryRow(ctx, `
		SELECT id, user_id, problem, solver_type, config, status,
		       best_fitness, best_individual, history, total_generations,
		       result_meta, error, created_at, updated_at, completed_at
		FROM jobs
		WHERE id = $1
	`, id).Scan(
		&j.ID, &j.UserID, &j.Problem, &j.SolverType, &configJSON, &j.Status,
		&j.BestFitness, &bestIndJSON, &historyJSON, &j.TotalGenerations,
		&metaJSON, &j.Error, &createdAt, &updatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSONB columns into Go values
	if configJSON != nil {
		json.Unmarshal(configJSON, &j.Config)
	}
	if bestIndJSON != nil {
		json.Unmarshal(bestIndJSON, &j.BestIndividual)
	}
	if historyJSON != nil {
		json.Unmarshal(historyJSON, &j.History)
	}
	if metaJSON != nil {
		json.Unmarshal(metaJSON, &j.ResultMeta)
	}

	return &j, nil
}

// GetJobForUser returns a job only if the caller owns it.
func (d *DB) GetJobForUser(ctx context.Context, jobID, userID string) (*JobRecord, error) {
	var j JobRecord
	var configJSON, bestIndJSON, historyJSON, metaJSON []byte
	var createdAt, updatedAt time.Time
	var completedAt *time.Time
	var ownerID *string

	err := d.pool.QueryRow(ctx, `
		SELECT id, user_id, problem, solver_type, config, status,
		       best_fitness, best_individual, history, total_generations,
		       result_meta, error, created_at, updated_at, completed_at
		FROM jobs
		WHERE id = $1 AND user_id = $2
	`, jobID, userID).Scan(
		&j.ID, &ownerID, &j.Problem, &j.SolverType, &configJSON, &j.Status,
		&j.BestFitness, &bestIndJSON, &historyJSON, &j.TotalGenerations,
		&metaJSON, &j.Error, &createdAt, &updatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	j.UserID = ownerID

	if configJSON != nil {
		json.Unmarshal(configJSON, &j.Config)
	}
	if bestIndJSON != nil {
		json.Unmarshal(bestIndJSON, &j.BestIndividual)
	}
	if historyJSON != nil {
		json.Unmarshal(historyJSON, &j.History)
	}
	if metaJSON != nil {
		json.Unmarshal(metaJSON, &j.ResultMeta)
	}
	return &j, nil
}
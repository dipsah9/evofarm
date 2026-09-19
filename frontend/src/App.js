import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { Line } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js';
import './App.css';
import JobHistory from './JobHistory';
import ScheduleGrid from './ScheduleGrid';

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
);



const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

function App() {
  const [activeTab, setActiveTab] = useState('submit'); // 'submit' | 'history'
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState(null);
//   const [formData, setFormData] = useState({
//     population_size: 100,
//     generations: 50,
//     fitness_function: 'xor',
//     solver_type: 'evolution',
//     problem: 'nurse_rostering',
//     time_limit_seconds: 30
//   });
    const [formData, setFormData] = useState({
    problem: 'xor',
    population_size: 100,
    generations: 50,
    });



  const [problems, setProblems] = useState({});

    useEffect(() => {
    const fetchCatalog = async () => {
        try {
        const res = await axios.get(`${API_URL}/problems`);
        setProblems(res.data);
        } catch (err) {
        console.error('Failed to load problem catalog:', err);
        }
    };
    fetchCatalog();
    }, []);

  // Submit a new job
  const submitJob = async (e) => {
  e.preventDefault();
  setLoading(true);
  try {
    const meta = problems[formData.problem] || {};
    const payload = { problem: formData.problem };

    if (meta.solver === 'cpsat') {
      payload.config = { num_nurses: 6, num_days: 7 };
      payload.time_limit_seconds = 30;
    } else {
      payload.population_size = formData.population_size;
      payload.generations = formData.generations;
    }

    const response = await axios.post(`${API_URL}/jobs`, payload);

    const newJob = {
      id: response.data.job_id,
      status: 'pending',
      progress: 0,
      best_fitness: 0,
      created_at: new Date().toISOString(),
    };
    setJobs([newJob, ...jobs]);
    setSelectedJobId(newJob.id);
  } catch (error) {
    console.error('Error submitting job:', error);
    alert('Failed to submit job. Is the API running?');
  } finally {
    setLoading(false);
  }
};

  const handleProblemChange = (problem) => {
    const meta = problems[problem] || {};
    const defaults = meta.solver === 'cpsat'
      ? { population_size: 0, generations: 0 }
      : { population_size: 100, generations: 50 };

    setFormData((prev) => ({
      ...prev,
      problem,
      ...defaults,
    }));
  };

  return (
    <div className="App">
      <header className="header">
        <div className="header-content">
          <h1>🧬 EvoFarm</h1>
          <p>Neuroevolution-as-a-Service</p>
        </div>
      </header>

      <div className="container">
        {/* Left Column: Submit Form + Job List */}
                <div className="sidebar">
        <div className="tab-bar">
            <button
            className={`tab ${activeTab === 'submit' ? 'active' : ''}`}
            onClick={() => setActiveTab('submit')}
            >
            New Job
            </button>
            <button
            className={`tab ${activeTab === 'history' ? 'active' : ''}`}
            onClick={() => setActiveTab('history')}
            >
            History
            </button>
        </div>

        {activeTab === 'submit' && (
            <>
            <div className="card">
                <h2> Submit Job</h2>
                <form onSubmit={submitJob}>
                {/* ... existing form content ... */}
                <div className="form-group">
                <label>Problem</label>
                <select
                value={formData.problem}
                onChange={(e) => handleProblemChange(e.target.value)}
                >
                {Object.keys(problems).length === 0 && (
                    <option value="xor">Loading...</option>
                )}
                {Object.entries(problems).map(([key, meta]) => (
                    <option key={key} value={key}>
                    {key} — {meta.description} ({meta.solver})
                    </option>
                ))}
                </select>
            </div>

            {/* Only show population/generations for evolution problems */}
            {problems[formData.problem]?.solver === 'evolution' && (
                <>
                <div className="form-group">
                    <label>Population Size</label>
                    <input
                    type="number"
                    value={formData.population_size}
                    onChange={(e) => setFormData({...formData, population_size: parseInt(e.target.value)})}
                    min="10"
                    max="1000"
                    required
                    />
                </div>
                <div className="form-group">
                    <label>Generations</label>
                    <input
                    type="number"
                    value={formData.generations}
                    onChange={(e) => setFormData({...formData, generations: parseInt(e.target.value)})}
                    min="5"
                    max="500"
                    required
                    />
                </div>
                </>
            )}

            <button type="submit" disabled={loading}>
                {loading ? 'Submitting...' : ' Start Job'}
            </button>
                </form>
            </div>

            <div className="card">
                <h2>📋 Recent Jobs</h2>
                <div className="job-list">
                {jobs.length === 0 ? (
                    <p className="empty-state">No jobs yet. Submit one above!</p>
                ) : (
                    jobs.map((job) => (
                    <JobListItem
                        key={job.id}
                        jobId={job.id}
                        initialStatus={job.status}
                        isSelected={selectedJobId === job.id}
                        onClick={() => setSelectedJobId(job.id)}
                    />
                    ))
                )}
                </div>
            </div>
            </>
        )}

        {activeTab === 'history' && (
            <div className="card">
            <h2>📚 Job History</h2>
            <JobHistory onSelectJob={(id) => {
                setSelectedJobId(id);
                //setActiveTab('submit');
            }} />
            </div>
        )}
        </div>

        {/* Right Column: Selected Job Detail with Chart */}
        <div className="main">
          {selectedJobId ? (
            <JobDetail jobId={selectedJobId} />
          ) : (
            <div className="card empty-detail">
              <h2>👈 Select a job to view details</h2>
              <p>Submit a new job or click one from the list to see live evolution progress.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Compact job item in the sidebar
function JobListItem({ jobId, initialStatus, isSelected, onClick }) {
  const [job, setJob] = useState({
    status: initialStatus || 'pending',
    progress: 0,
    best_fitness: 0,
    total_generations: 0 
  });

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await axios.get(`${API_URL}/jobs/${jobId}`);
        setJob({
          status: response.data.status,
          progress: response.data.progress || 0,
          best_fitness: response.data.best_fitness || 0,
           total_generations: response.data.total_generations || 0
        });
      } catch (error) {
        console.error('Error:', error);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 2000);
    return () => clearInterval(interval);
  }, [jobId]);

  const getStatusIcon = () => {
    switch (job.status) {
      case 'completed': return '✅';
      case 'running': return '🔄';
      case 'failed': return '❌';
      default: return '⏳';
    }
  };


  return (
    <div
      className={`job-list-item ${isSelected ? 'selected' : ''}`}
      onClick={onClick}
    >
      <div className="job-list-header">
        <span className="job-id">{jobId.substring(0, 8)}...</span>
        <span className="status-icon">{getStatusIcon()}</span>
      </div>
      <div className="progress-bar-small">
        <div
          className="progress-fill-small"
          style={{ width: `${job.progress * 100}%` }}
        ></div>
      </div>
      <div className="job-list-footer">
        <span className="fitness-small">
          {job.best_fitness ? job.best_fitness.toFixed(3) : '—'}
        </span>
        <span className="status-text-small">{job.status}</span>
      </div>
    </div>
  );
}

function JobDetail({ jobId }) {
  const [job, setJob] = useState({
    status: 'pending',
    progress: 0,
    best_fitness: 0,
    best_individual: [],
    error: '',
    total_generations: 0,
    result_meta: {}
  });
  const [history, setHistory] = useState([]);

  useEffect(() => {
    setHistory([]);

    const fetchJobStatus = async () => {
      try {
        const response = await axios.get(`${API_URL}/jobs/${jobId}`);
        const data = response.data;

        setJob({
          status: data.status,
          progress: data.progress || 0,
          best_fitness: data.best_fitness || 0,
          best_individual: data.best_individual || [],
          error: data.error || '',
          total_generations: data.total_generations || 0,
          result_meta: data.result_meta || {}
        });

        if (data.history && data.history.length > 0) {
          setHistory(data.history);
        }
      } catch (error) {
        console.error('Error fetching job:', error);
      }
    };

    fetchJobStatus();
    const interval = setInterval(fetchJobStatus, 2000);
    return () => clearInterval(interval);
  }, [jobId]);

  const getStatusColor = () => {
    switch (job.status) {
      case 'completed': return 'status-completed';
      case 'running':   return 'status-running';
      case 'failed':    return 'status-failed';
      default:          return 'status-pending';
    }
  };

  const isCPSAT = job.result_meta && job.result_meta.solver === 'cpsat';
  const isEvolution = !isCPSAT;

  // ---------- Chart data (evolution only) ----------
  const chartData = {
    labels: history.map(h => `Gen ${h.generation}`),
    datasets: [{
      label: 'Best Fitness',
      data: history.map(h => h.fitness),
      borderColor: '#667eea',
      backgroundColor: 'rgba(102, 126, 234, 0.15)',
      fill: true,
      tension: 0.4,
      pointBackgroundColor: '#764ba2',
      pointBorderColor: '#fff',
      pointBorderWidth: 2,
      pointRadius: 4,
      pointHoverRadius: 6
    }]
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: '#141a2e',
        titleColor: '#ccd6f6',
        bodyColor: '#e0e0e0',
        borderColor: '#667eea',
        borderWidth: 1,
        padding: 12,
        displayColors: false
      }
    },
    scales: {
      x: {
        grid: { color: '#1a1f35' },
        ticks: {
          color: '#8892b0',
          maxRotation: 45,
          minRotation: 45,
          autoSkip: true,
          maxTicksLimit: 15
        }
      },
      y: {
        min: 0,
        max: 1,
        grid: { color: '#1a1f35' },
        ticks: { color: '#8892b0' }
      }
    }
  };

  return (
    <div className="card job-detail">
      <div className="detail-header">
        <div>
          <h2>Job Details</h2>
          <span className="job-id-full">{jobId}</span>
        </div>
        <div className="detail-header-right">
          {isCPSAT && <span className="solver-badge solver-cpsat">CP-SAT</span>}
          {isEvolution && <span className="solver-badge solver-evolution">Evolution</span>}
          <span className={`status-badge ${getStatusColor()}`}>
            {job.status}
          </span>
        </div>
      </div>

      {/* ---------- Stats ---------- */}
      {isEvolution && (
        <div className="stats-grid">
          <div className="stat">
            <label>Best Fitness</label>
            <div className="stat-value">
              {job.best_fitness ? job.best_fitness.toFixed(4) : '—'}
            </div>
          </div>
          <div className="stat">
            <label>Progress</label>
            <div className="stat-value">{(job.progress * 100).toFixed(0)}%</div>
          </div>
          <div className="stat">
            <label>Generations</label>
            <div className="stat-value">{job.total_generations || history.length}</div>
          </div>
        </div>
      )}

      {isCPSAT && (
        <div className="stats-grid">
          <div className="stat">
            <label>Solver Status</label>
            <div className="stat-value">{job.result_meta.status || '—'}</div>
          </div>
          <div className="stat">
            <label>Solve Time</label>
            <div className="stat-value">
              {job.result_meta.solve_time_seconds != null
                ? `${job.result_meta.solve_time_seconds}s`
                : '—'}
            </div>
          </div>
          <div className="stat">
            <label>Objective</label>
            <div className="stat-value">
              {job.best_fitness != null ? job.best_fitness.toFixed(0) : '0'}
            </div>
          </div>
        </div>
      )}

      <div className="progress-bar">
        <div className="progress-fill" style={{ width: `${job.progress * 100}%` }}></div>
      </div>

      {/* ---------- Body: Chart or Schedule ---------- */}
      {isEvolution && (
        <>
          <h3 className="section-title">📈 Fitness Over Generations</h3>
          <div className="chart-container">
            {history.length > 0 ? (
              <Line data={chartData} options={chartOptions} />
            ) : (
              <div className="chart-empty">
                {job.status === 'completed'
                  ? 'Job completed (chart data may be partial)'
                  : 'Waiting for evolution to start...'}
              </div>
            )}
          </div>
        </>
      )}

      {isCPSAT && (
        <>
          <h3 className="section-title">
            📅 Nurse Schedule
            {job.result_meta.num_nurses && job.result_meta.num_days && (
              <span className="section-subtitle">
                {' '}({job.result_meta.num_nurses} nurses × {job.result_meta.num_days} days)
              </span>
            )}
          </h3>
          <ScheduleGrid schedule={job.best_individual} />
        </>
      )}

      {/* ---------- Evolution weights ---------- */}
      {/* Portfolio job: 8 weights → render allocation bars */}
        {isEvolution && job.status === 'completed' &&
        Array.isArray(job.best_individual) &&
        job.best_individual.length === 8 &&
        job.best_individual.every(v => typeof v === 'number') && (
            <PortfolioDisplay rawGenome={job.best_individual} />
        )}

        {/* XOR job: 17 weights → render weight tags */}
        {isEvolution && job.status === 'completed' &&
        Array.isArray(job.best_individual) &&
        job.best_individual.length === 17 &&
        job.best_individual.every(v => typeof v === 'number') && (
            <div className="result-section">
            <h3 className="section-title">🧠 Evolved Network</h3>
            <div className="weights-display">
                {job.best_individual.slice(0, 8).map((w, i) => (
                <span key={i} className="weight-tag">{w.toFixed(3)}</span>
                ))}
                <span className="weight-tag more">
                +{job.best_individual.length - 8} more
                </span>
            </div>
            </div>
        )}
      {job.status === 'failed' && (
        <div className="error-message">
          ⚠️ {job.error || 'Job failed. Check worker logs.'}
        </div>
      )}
    </div>
  );
}

function PortfolioDisplay({ rawGenome }) {
  const ASSETS = [
    'US Equity',
    'EU Equity',
    'Emerging Markets',
    'Government Bonds',
    'Corporate Bonds',
    'Gold',
    'Real Estate',
    'Commodities',
  ];

  // Must match the backend softmax exactly
  const maxVal = Math.max(...rawGenome);
  const exps = rawGenome.map(v => Math.exp(v - maxVal));
  const sumExp = exps.reduce((a, b) => a + b, 0);
  const weights = exps.map(v => v / sumExp);

  const COLORS = [
    '#667eea', '#764ba2', '#8fa4ff', '#b794f6',
    '#4caf50', '#f4b942', '#e67e22', '#e74c3c',
  ];

  // Sort by weight descending for readability
  const rows = weights
    .map((w, i) => ({ name: ASSETS[i], weight: w, color: COLORS[i] }))
    .sort((a, b) => b.weight - a.weight);

  return (
    <div className="result-section">
      <h3 className="section-title">💼 Portfolio Allocation</h3>
      <div className="portfolio-bars">
        {rows.map((row, i) => (
          <div key={i} className="portfolio-row">
            <div className="portfolio-label">{row.name}</div>
            <div className="portfolio-bar-track">
              <div
                className="portfolio-bar-fill"
                style={{
                  width: `${(row.weight * 100).toFixed(2)}%`,
                  background: row.color,
                }}
              ></div>
            </div>
            <div className="portfolio-value">
              {(row.weight * 100).toFixed(1)}%
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default App;
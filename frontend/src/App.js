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
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState(null);
  const [formData, setFormData] = useState({
    population_size: 100,
    generations: 50,
    fitness_function: 'xor'
  });

  // Submit a new job
  const submitJob = async (e) => {
    e.preventDefault();
    setLoading(true);
    try {
      const response = await axios.post(`${API_URL}/jobs`, formData);
      const newJob = {
        id: response.data.job_id,
        status: 'pending',
        progress: 0,
        best_fitness: 0,
        created_at: new Date().toISOString()
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
          <div className="card">
            <h2>Submit Job</h2>
            <form onSubmit={submitJob}>
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
              <div className="form-group">
                <label>Fitness Function</label>
                <select
                  value={formData.fitness_function}
                  onChange={(e) => setFormData({...formData, fitness_function: e.target.value})}
                >
                  <option value="xor">XOR</option>
                </select>
              </div>
              <button type="submit" disabled={loading}>
                {loading ? 'Submitting...' : ' Start Evolution'}
              </button>
            </form>
          </div>

          <div className="card">
            <h2>📋 Jobs</h2>
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

// Detailed view with chart
function JobDetail({ jobId }) {
  const [job, setJob] = useState({
    status: 'pending',
    progress: 0,
    best_fitness: 0,
    best_individual: [],
    error: ''
  });
  const [history, setHistory] = useState([]);

  useEffect(() => {
    // Reset history when job changes
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
          error: data.error || ''
        });

        if (data.history && data.history.length > 0) {
        setHistory(data.history);
        }

        // Track fitness history when the API does not provide historical data.
        if (data.best_fitness > 0 && data.status === 'running') {
          setHistory(prev => {
            if (prev.length > 0 && prev[prev.length - 1].fitness === data.best_fitness) {
              return prev;
            }
            return [...prev, {
              generation: prev.length + 1,
              fitness: data.best_fitness
            }];
          });
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
      case 'running': return 'status-running';
      case 'failed': return 'status-failed';
      default: return 'status-pending';
    }
  };

  // Chart data
  const chartData = {
    labels: history.map(h => `Gen ${h.generation}`),
    datasets: [
      {
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
      }
    ]
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: false
      },
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
        <span className={`status-badge ${getStatusColor()}`}>
          {job.status}
        </span>
      </div>

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

      <div className="progress-bar">
        <div
          className="progress-fill"
          style={{ width: `${job.progress * 100}%` }}
        ></div>
      </div>

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

      {job.status === 'completed' && job.best_individual.length > 0 && (
        <div className="result-section">
          <h3 className="section-title">🧠 Evolved Network</h3>
          <div className="weights-display">
            {job.best_individual.slice(0, 8).map((w, i) => (
              <span key={i} className="weight-tag">
                {w.toFixed(3)}
              </span>
            ))}
            {job.best_individual.length > 8 && (
              <span className="weight-tag more">
                +{job.best_individual.length - 8} more
              </span>
            )}
          </div>
        </div>
      )}

      {job.status === 'failed' && (
        <div className="error-message">
          ⚠️ {job.error || 'Evolution failed. Check worker logs.'}
        </div>
      )}
    </div>
  );
}

export default App;